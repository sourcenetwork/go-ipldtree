// MIT License
//
// # Copyright (c) 2023 Tommy TIAN
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

	"github.com/agiledragon/gomonkey/v2"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

func setupTestVerify(size int) (*IPLDTree, []blocks.Block) {
	blocks := mockDataBlocks(size)
	ctx := context.Background()
	m, err := New(ctx, nil, newTestBlockstore(), blocks)
	if err != nil {
		panic(err)
	}
	return m, blocks
}

// func setupTestVerifyParallel(size int) (*IPLDTree, []DataBlock) {
// 	blocks := mockDataBlocks(size)
// 	m, err := New(&Config{
// 		RunInParallel: true,
// 		NumRoutines:   1,
// 	}, blocks)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return m, blocks
// }

func TestIPLDTreeVerify(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(int) (*IPLDTree, []blocks.Block)
		blockSize int
		want      bool
		wantErr   bool
	}{
		{
			name:      "test_2",
			setupFunc: setupTestVerify,
			blockSize: 2,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "test_3",
			setupFunc: setupTestVerify,
			blockSize: 3,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "test_4",
			setupFunc: setupTestVerify,
			blockSize: 4,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "test_5",
			setupFunc: setupTestVerify,
			blockSize: 5,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "test_6",
			setupFunc: setupTestVerify,
			blockSize: 6,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "test_8",
			setupFunc: setupTestVerify,
			blockSize: 8,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "test_9",
			setupFunc: setupTestVerify,
			blockSize: 9,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "test_1001",
			setupFunc: setupTestVerify,
			blockSize: 1001,
			want:      true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, blocks := tt.setupFunc(tt.blockSize)
			for i := 0; i < tt.blockSize; i++ {
				got, err := m.Verify(blocks[i], m.proofs[i])
				if (err != nil) != tt.wantErr {
					t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("Verify() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestVerify(t *testing.T) {
	blks := mockDataBlocks(5)
	ctx := context.Background()
	m, err := New(ctx, nil, newTestBlockstore(), blks)
	if err != nil {
		t.Errorf("New() error = %v", err)
		return
	}
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	type args struct {
		dataBlock blocks.Block
		proof     *Proof
		root      cid.Cid
	}
	tests := []struct {
		name    string
		args    args
		mock    func()
		want    bool
		wantErr bool
	}{
		{
			name: "test_ok",
			args: args{
				dataBlock: blks[0],
				proof:     m.proofs[0],
				root:      m.root.Data.Cid(),
			},
			want: true,
		},
		{
			name: "test_wrong_root",
			args: args{
				dataBlock: blks[0],
				proof:     m.proofs[0],
				root:      cid.MustParse("bafyreib4hmpkwa7zyzoxmpwykof6k7akxnvmsn23oiubsey4e2tf6gqlui"),
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "test_proof_nil",
			args: args{
				dataBlock: blks[0],
				proof:     nil,
				root:      m.root.Data.Cid(),
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "test_data_block_nil",
			args: args{
				dataBlock: nil,
				proof:     m.proofs[0],
				root:      m.root.Data.Cid(),
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "test_hash_func_nil",
			args: args{
				dataBlock: blks[0],
				proof:     m.proofs[0],
				root:      m.root.Data.Cid(),
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil {
				tt.mock()
			}
			defer patches.Reset()
			got, err := Verify(ctx, m.blockstore, tt.args.dataBlock, tt.args.proof, tt.args.root, m.Config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyPathOnly(t *testing.T) {
	blks := mockDataBlocks(64)
	bstore := newTestBlockstore()
	bservice := newBlockservice(bstore)
	ctx := context.Background()
	m, err := New(ctx, nil, bstore, blks)
	if err != nil {
		t.Errorf("New() error = %v", err)
		return
	}
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	type args struct {
		dataBlock blocks.Block
		proof     *Proof
		root      cid.Cid
	}
	tests := []struct {
		name    string
		args    args
		mock    func()
		want    bool
		wantErr bool
	}{
		{
			name: "test_ok",
			args: args{
				dataBlock: blks[42],
				proof:     m.proofs[42],
				root:      m.root.Data.Cid(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil {
				tt.mock()
			}
			defer patches.Reset()
			got, err := Verify(ctx, bservice, tt.args.dataBlock, tt.args.proof, tt.args.root, m.Config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}
