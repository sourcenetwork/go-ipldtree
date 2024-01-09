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

package merkletree

import (
	"context"
	"fmt"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"
)

// Verify checks if the data block is valid using the Merkle Tree proof and the cached Merkle root hash.
func (m *IPLDTree) Verify(dataBlock blocks.Block, proof *Proof) (bool, error) {
	return Verify(m.ctx, m.blockstore, dataBlock, proof, m.root.Data.Cid(), m.Config)
}

// VerifyWithPath checks if the data block is valid using the Merkle Tree proof and the provided root hash.
// It differs from Verify() because it relies on the blockstore to resolve the sibling nodes.
func Verify(ctx context.Context, bs Blockstore, dataBlock blocks.Block, proof *Proof, root cid.Cid, config Config) (bool, error) {
	if proof == nil {
		return false, ErrProofIsNil
	}

	if dataBlock == nil {
		return false, ErrDataBlockIsNil
	}

	result, err := dataBlockToLeaf(dataBlock, nil, config.DisableLeafHashing)
	if err != nil {
		return false, err
	}

	path := proof.Path
	nd, err := getNodeFromBlockstore(ctx, root, bs)
	if err != nil {
		return false, fmt.Errorf("root from blockstore: %w", err)
	}

	// get sibs
	bit := uint32(1 << (proof.Len - 1))
	var link cid.Cid
	for i := 0; i < int(proof.Len); i++ {
		if path&bit > 0 {
			links := nd.Links()
			if len(links) != 2 {
				return false, fmt.Errorf("bad ipld merkle node")
			}
			link = links[0].Cid
		} else {
			links := nd.Links()
			if len(links) != 2 {
				return false, fmt.Errorf("bad ipld merkle node")
			}
			link = links[1].Cid
		}

		bit >>= 1
		// follow the path through the blockstore

		// peek ahead, if theres another iteration, get the block
		if i+1 < int(proof.Len) {
			nd, err = getNodeFromBlockstore(ctx, link, bs)
			if err != nil {
				return false, fmt.Errorf("next sibling node from blockstore: %w", err)
			}
		}
	}

	return result.Data.Cid().Equals(link), nil
}

func getNodeFromBlockstore(ctx context.Context, c cid.Cid, bs Blockstore) (*cbor.Node, error) {
	blk, err := bs.Get(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("blockstore missing block %s: %w", c, err)
	}

	nd, err := cbor.Decode(blk.RawData(), mh.SHA2_256, -1)
	if err != nil {
		return nil, err
	}

	return nd, nil
}

// Verify checks if the data block is valid using the Merkle Tree proof and the provided Merkle root hash.
// It returns true if the data block is valid, false otherwise. An error is returned in case of any issues
// during the verification process.
/*func Verify(ctx context.Context, dataBlock blocks.Block, proof *Proof, root cid.Cid, config Config) (bool, error) {
	if proof == nil {
		return false, ErrProofIsNil
	}

	if dataBlock == nil {
		return false, fmt.Errorf("invalid datablock")
	}

	if config.MergeFunc == nil {
		return false, fmt.Errorf("missing merge func")
	}

	// Determine the concatenation function based on the configuration.
	// concatFunc := concatHash
	// if config.SortSiblingPairs {
	// 	concatFunc = concatSortHash
	// }

	// Convert the data block to a leaf.
	result, err := dataBlockToLeaf(dataBlock, nil, config.DisableLeafHashing)
	if err != nil {
		return false, err
	}

	// Traverse the Merkle proof and compute the resulting hash.
	// Copy the slice so that the original leaf won't be modified.
	// result := make([]byte, len(leaf))
	// copy(result, leaf)

	path := proof.Path

	for _, sib := range proof.Siblings {
		if path&1 == 1 {
			fmt.Println("left:", result)
			fmt.Println("right:", sib)
			result, err = config.MergeFunc(ctx, result, sib)
		} else {
			result, err = config.MergeFunc(ctx, sib, result)
		}

		if err != nil {
			return false, err
		}

		path >>= 1
	}

	return result.Data.Cid().Equals(root), nil
}*/
