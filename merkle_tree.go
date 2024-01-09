// MIT License
//
// Copyright (c) 2023 Tommy TIAN
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package merkletree implements a high-performance Merkle Tree in Go.
// It supports parallel execution for enhanced performance and
// offers compatibility with OpenZeppelin through sorted sibling pairs.
package merkletree

import (
	"bytes"
	"context"
	"fmt"
	"math/bits"

	"github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"
)

const (
	// ModeProofGen is the proof generation configuration mode.
	ModeProofGen TypeConfigMode = iota
	// ModeTreeBuild is the tree building configuration mode.
	ModeTreeBuild
	// ModeProofGenAndTreeBuild is the proof generation and tree building configuration mode.
	ModeProofGenAndTreeBuild
)

// TypeConfigMode is the type in the Merkle Tree configuration indicating what operations are performed.
type TypeConfigMode int

// TypeHashFunc is the signature of the hash functions used for Merkle Tree generation.
type TypeHashFunc func([]byte) ([]byte, error)

type typeConcatHashFunc func([]byte, []byte) []byte

// Config is the configuration of Merkle Tree.
type Config struct {
	// Customizable hash function used for tree generation.
	//HashFunc TypeHashFunc

	// MergeFunc
	MergeFunc MergeFunc

	// CompareFunc
	// CompareFunc CompareFunc

	// // ExtFunc
	// ExtFunc ExtFunc

	// Number of goroutines run in parallel.
	// If RunInParallel is true and NumRoutine is set to 0, use number of CPU as the number of goroutines.
	//NumRoutines int

	// Mode of the Merkle Tree generation.
	Mode TypeConfigMode

	// If RunInParallel is true, the generation runs in parallel, otherwise runs without parallelization.
	// This increase the performance for the calculation of large number of data blocks, e.g. over 10,000 blocks.
	//RunInParallel bool

	// SortSiblingPairs is the parameter for OpenZeppelin compatibility.
	// If set to `true`, the hashing sibling pairs are sorted.
	SortSiblingPairs bool

	// If true, the leaf nodes are NOT hashed before being added to the Merkle Tree.
	DisableLeafHashing bool
}

type Contentable interface {
	Cid() cid.Cid
}

type MergeFunc func(ctx context.Context, left, right Node) (Node, error)

type Blockstore interface {
	Put(context.Context, any) (blocks.Block, error)
	Get(context.Context, cid.Cid) (blocks.Block, error)
}

func ipldMerge(blockstore Blockstore) MergeFunc {
	return func(ctx context.Context, left, right Node) (Node, error) {
		inner := []cid.Cid{left.Data.Cid(), right.Data.Cid()}
		data, err := blockstore.Put(ctx, inner)
		return Node{Data: data, Left: &left, Right: &right}, err
	}
}

func ipldCompare(a, b Contentable) int {
	return bytes.Compare(a.Cid().Bytes(), b.Cid().Bytes())
}

type CompareFunc[T any] func(a, b T) int

// type ExtFunc[T any, S any] func(Node[T]) S

type Node struct {
	Data  blocks.Block
	Left  *Node
	Right *Node
}

type Result interface {
	String() string
	Equals(Result) bool
}

// IPLDTree implements the Merkle Tree data structure.
type IPLDTree struct {
	Config

	ctx context.Context

	blockstore Blockstore

	// leafMap maps the data (converted to string) of each leaf node to its index in the Tree slice.
	// It is only available when the configuration mode is set to ModeTreeBuild or ModeProofGenAndTreeBuild.
	leafMap map[string]int

	// concatHashFunc is the function for concatenating two hashes.
	// If SortSiblingPairs in Config is true, then the sibling pairs are first sorted and then concatenated,
	// supporting the OpenZeppelin Merkle Tree protocol.
	// Otherwise, the sibling pairs are concatenated directly.
	// mergeFn MergeFunc[T]

	// nodes contains the Merkle Tree's internal node structure.
	// It is only available when the configuration mode is set to ModeTreeBuild or ModeProofGenAndTreeBuild.
	nodes [][]Node

	// root is the hash of the Merkle root node.
	root Node

	// Leaves are the hashes of the data blocks that form the Merkle Tree's leaves.
	// These hashes are used to generate the tree structure.
	// If the DisableLeafHashing configuration is set to true, the original data blocks are used as the leaves.
	leaves []Node

	// Proofs are the proofs to the data blocks generated during the tree building process.
	proofs []*Proof

	// Depth is the depth of the Merkle Tree.
	depth int

	// NumLeaves is the number of leaves in the Merkle Tree.
	// This value is fixed once the tree is built.
	numLeaves int
}

// New generates a new Merkle Tree with the specified configuration and data blocks.
func New(ctx context.Context, config *Config, blockstore blockstore.Blockstore, blocks []blocks.Block) (m *IPLDTree, err error) {
	// Check if there are enough data blocks to build the tree.
	if len(blocks) <= 1 {
		return nil, ErrInvalidNumOfDataBlocks
	}

	if blockstore == nil {
		return nil, fmt.Errorf("missing blockstore")
	}

	blockservice := newBlockservice(blockstore)

	// // Initialize the configuration if it is not provided.
	if config == nil {
		config = new(Config)
	}

	if config.MergeFunc == nil {
		config.MergeFunc = ipldMerge(blockservice)
	}

	// Create a IPLDTree with the provided configuration.
	m = &IPLDTree{
		ctx:        ctx,
		blockstore: blockservice,
		Config:     *config,
		numLeaves:  len(blocks),
		depth:      bits.Len(uint(len(blocks) - 1)),
	}

	// Hash concatenation function initialization.
	// if m.concatHashFunc == nil {
	// 	if m.SortSiblingPairs {
	// 		m.concatHashFunc = concatSortHash
	// 	} else {
	// 		m.concatHashFunc = concatHash
	// 	}
	// }

	// Perform actions based on the configured mode.
	// Set the mode to ModeProofGen by default if not specified.
	if m.Mode == 0 {
		m.Mode = ModeProofGen
	}

	if err := m.new(blocks); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *IPLDTree) new(blocks []blocks.Block) error {
	// Initialize the hash function.
	if m.MergeFunc == nil {
		panic("hashfunc nil")
	}
	/* TODO
	if m.CompareFunc == nil {
		panic("comparefunc nil")
	}
	if m.ExtFunc == nil {
		panic("extfunc nil")
	}
	TODO */

	// Generate leaves.
	var err error
	m.leaves, err = m.computeLeafNodes(blocks)

	if err != nil {
		return err
	}

	// Initialize the leafMap for ModeTreeBuild and ModeProofGenAndTreeBuild.
	m.leafMap = make(map[string]int)

	return m.proofGen()

	/* TODO
	if m.Mode == ModeTreeBuild {
		return m.treeBuild()
	}

	// Build the tree and generate proofs in ModeProofGenAndTreeBuild.
	if m.Mode == ModeProofGenAndTreeBuild {
		return m.proofGenAndTreeBuild()
	}

	// Return an error if the configuration mode is invalid.
	return ErrInvalidConfigMode
	TODO */

}

// func (m *IPLDTree[T, S]) newParallel(blocks []DataBlock) error {
// 	// Initialize the hash function.
// 	if m.HashFunc == nil {
// 		m.HashFunc = DefaultHashFuncParallel
// 	}

// 	// Set NumRoutines to the number of CPU cores if not specified or invalid.
// 	if m.NumRoutines <= 0 {
// 		m.NumRoutines = runtime.NumCPU()
// 	}

// 	// Generate leaves.
// 	var err error
// 	m.Leaves, err = m.computeLeafNodesParallel(blocks)

// 	if err != nil {
// 		return err
// 	}

// 	if m.Mode == ModeProofGen {
// 		return m.proofGenParallel()
// 	}

// 	// Initialize the leafMap for ModeTreeBuild and ModeProofGenAndTreeBuild.
// 	m.leafMap = make(map[string]int)

// 	if m.Mode == ModeTreeBuild {
// 		return m.treeBuildParallel()
// 	}

// 	// Build the tree and generate proofs in ModeProofGenAndTreeBuild.
// 	if m.Mode == ModeProofGenAndTreeBuild {
// 		return m.proofGenAndTreeBuildParallel()
// 	}

// 	// Return an error if the configuration mode is invalid.
// 	return ErrInvalidConfigMode
// }

// concatHash concatenates two byte slices, b1 and b2.
func concatHash(b1, b2 []byte) []byte {
	result := make([]byte, len(b1)+len(b2))
	copy(result, b1)
	copy(result[len(b1):], b2)

	return result
}

// concatSortHash concatenates two byte slices, b1 and b2, in a  sorted order.
// The function ensures that the smaller byte slice (in terms of lexicographic order)
// is placed before the larger one. This is used for compatibility with OpenZeppelin's
// Merkle Proof verification implementation.
func concatSortHash(b1, b2 []byte) []byte {
	if bytes.Compare(b1, b2) < 0 {
		return concatHash(b1, b2)
	}

	return concatHash(b2, b1)
}

var _ Blockstore = (*blockservice)(nil)

type blockservice struct {
	bs blockstore.Blockstore
}

func newBlockservice(bs blockstore.Blockstore) blockservice {
	return blockservice{bs}
}

func (bs blockservice) Put(ctx context.Context, obj any) (blocks.Block, error) {
	var blk blocks.Block
	var err error
	if b, ok := obj.(blocks.Block); ok {
		blk = b
	} else {
		blk, err = cbor.WrapObject(obj, mh.SHA2_256, -1)
		if err != nil {
			return nil, err
		}
	}

	err = bs.bs.Put(ctx, blk)
	return blk, err
}

func (bs blockservice) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	return bs.bs.Get(ctx, c)
}
