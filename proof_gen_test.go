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
	"testing"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

func TestIPLDTreeNew_modeProofGen(t *testing.T) {
	dummyDataBlocks := []blocks.Block{
		dataBlock([]byte("dummy_data_0")),
		dataBlock([]byte("dummy_data_1")),
		dataBlock([]byte("dummy_data_2")),
		dataBlock([]byte("dummy_data_3")),
		dataBlock([]byte("dummy_data_4")),
	}
	dummyRootSizeTwo, err := cid.Parse("bafyreif5eihvd4f7c6yyxyifv4hf2xmbqlwkvjgg3owd2b7jergfknif3i")
	if err != nil {
		t.Fatal(err)
	}
	dummyRootSizeThree, err := cid.Parse("bafyreihef2wmny3jtei3o2w4hju2nmhjesowoewouukojpmi4cvo6j6r7q")
	if err != nil {
		t.Fatal(err)
	}
	dummyRootSizeFour, err := cid.Parse("bafyreih55cz6q53ur2lovm2d6cl4n5rysntuv6huljadmzonciepaujwoi")
	if err != nil {
		t.Fatal(err)
	}
	dummyRootSizeFive, err := cid.Parse("bafyreiavkb2feikcnmrfefhcb3afhjx7tgalidll6nbjk6tqe3zot4zyuq")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		blocks []blocks.Block
		config *Config
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantRoot cid.Cid
	}{
		{
			name: "test_0",
			args: args{
				blocks: mockDataBlocks(0),
			},
			wantErr: true,
		},
		{
			name: "test_1",
			args: args{
				blocks: mockDataBlocks(1),
			},
			wantErr: true,
		},
		{
			name: "test_2",
			args: args{
				blocks: []blocks.Block{dummyDataBlocks[0], dummyDataBlocks[1]},
			},
			wantErr:  false,
			wantRoot: dummyRootSizeTwo,
		},
		{
			name: "test_3",
			args: args{
				blocks: []blocks.Block{dummyDataBlocks[0], dummyDataBlocks[1], dummyDataBlocks[2]},
			},
			wantErr:  false,
			wantRoot: dummyRootSizeThree,
		},
		{
			name: "test_4",
			args: args{
				blocks: []blocks.Block{dummyDataBlocks[0], dummyDataBlocks[1], dummyDataBlocks[2], dummyDataBlocks[3]},
			},
			wantErr:  false,
			wantRoot: dummyRootSizeFour,
		},
		{
			name: "test_5",
			args: args{
				blocks: dummyDataBlocks,
			},
			wantErr:  false,
			wantRoot: dummyRootSizeFive,
		},
		{
			name: "test_7",
			args: args{
				blocks: mockDataBlocks(7),
			},
			wantErr: false,
		},
		{
			name: "test_8",
			args: args{
				blocks: mockDataBlocks(8),
			},
			wantErr: false,
		},
		{
			name: "test_5",
			args: args{
				blocks: mockDataBlocks(5),
			},
			wantErr: false,
		},
		{
			name: "test_1000",
			args: args{
				blocks: mockDataBlocks(1000),
			},
			wantErr: false,
		},
		// {
		// 	name: "test_8_sorted",
		// 	args: args{
		// 		blocks: mockDataBlocks(8),
		// 		config: &Config{
		// 			SortSiblingPairs: true,
		// 		},
		// 	},
		// 	wantErr: false,
		// },
		// {
		// 	name: "test_hash_func_error",
		// 	args: args{
		// 		blocks: mockDataBlocks(100),
		// 		config: &Config{
		// 			HashFunc: func([]byte) ([]byte, error) {
		// 				return nil, fmt.Errorf("hash func error")
		// 			},
		// 		},
		// 	},
		// 	wantErr: true,
		// },
		// {
		// 	name: "test_100_disable_leaf_hashing",
		// 	args: args{
		// 		blocks: mockDataBlocks(100),
		// 		config: &Config{
		// 			DisableLeafHashing: true,
		// 		},
		// 	},
		// 	wantErr: false,
		// },
	}
	bs := newTestBlockstore()
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt, err := New(ctx, tt.args.config, bs, tt.args.blocks)
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if mt == nil {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantRoot == cid.Undef {
				for idx, block := range tt.args.blocks {
					ok, err := mt.Verify(block, mt.proofs[idx])
					if err != nil {
						t.Errorf("proof verification error, idx %d, err %v", idx, err)
						return
					}
					if !ok {
						t.Errorf("proof verification failed, idx %d", idx)
						return
					}
				}
			} else {
				if !mt.root.Data.Cid().Equals(tt.wantRoot) {
					t.Errorf("root mismatch, got %s, want %s", mt.root.Data.Cid(), tt.wantRoot)
					return
				}
			}

		})
	}
}

// func mockHashFunc(data []byte) ([]byte, error) {
// 	sha256Func := sha256.New()
// 	sha256Func.Write(data)
// 	return sha256Func.Sum(nil), nil
// }

// func TestIPLDTree_proofGen(t *testing.T) {
// 	patches := gomonkey.NewPatches()
// 	defer patches.Reset()
// 	type args struct {
// 		config *Config
// 		blocks []DataBlock
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		mock    func()
// 		wantErr bool
// 	}{
// 		{
// 			name: "test_hash_func_err",
// 			args: args{
// 				blocks: mockDataBlocks(5),
// 			},
// 			mock: func() {
// 				patches.ApplyFunc(mockHashFunc,
// 					func([]byte) ([]byte, error) {
// 						return nil, errors.New("test_hash_func_err")
// 					})
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			m, err := New(tt.args.config, tt.args.blocks)
// 			if err != nil {
// 				t.Errorf("New() error = %v", err)
// 				return
// 			}
// 			if tt.mock != nil {
// 				tt.mock()
// 			}
// 			defer patches.Reset()
// 			if err := m.proofGen(); (err != nil) != tt.wantErr {
// 				t.Errorf("proofGen() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

// func TestIPLDTree_proofGenParallel(t *testing.T) {
// 	var hashFuncCounter atomic.Uint32
// 	type args struct {
// 		config *Config
// 		blocks []DataBlock
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		wantErr bool
// 	}{
// 		{
// 			name: "test_goroutine_err",
// 			args: args{
// 				config: &Config{
// 					HashFunc: func(data []byte) ([]byte, error) {
// 						if hashFuncCounter.Load() == 9 {
// 							return nil, errors.New("test_goroutine_err")
// 						}
// 						hashFuncCounter.Add(1)
// 						return mockHashFunc(data)
// 					},
// 					RunInParallel: true,
// 				},
// 				blocks: mockDataBlocks(4),
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			m, err := New(tt.args.config, tt.args.blocks)
// 			if err != nil {
// 				t.Errorf("New() error = %v", err)
// 				return
// 			}
// 			if err := m.proofGenParallel(); (err != nil) != tt.wantErr {
// 				t.Errorf("proofGenParallel() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }
