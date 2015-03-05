package math

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/ethutil"
)

type Summer interface {
	Sum(i int) *big.Int
	Len() int
}

func Sum(slice Summer) (sum *big.Int) {
	sum = new(big.Int)

	for i := 0; i < slice.Len(); i++ {
		sum.Add(sum, slice.Sum(i))
	}
	return
}

type Vector struct {
	Gas, Price *big.Int
}

type VectorsBy func(v1, v2 Vector) bool

func (self VectorsBy) Sort(vectors []Vector) {
	bs := vectorSorter{
		vectors: vectors,
		by:      self,
	}
	sort.Sort(bs)
}

type vectorSorter struct {
	vectors []Vector
	by      func(v1, v2 Vector) bool
}

func (v vectorSorter) Len() int           { return len(v.vectors) }
func (v vectorSorter) Less(i, j int) bool { return v.by(v.vectors[i], v.vectors[j]) }
func (v vectorSorter) Swap(i, j int)      { v.vectors[i], v.vectors[j] = v.vectors[j], v.vectors[i] }

func PriceSort(v1, v2 Vector) bool { return v1.Price.Cmp(v2.Price) < 0 }
func GasSort(v1, v2 Vector) bool   { return v1.Gas.Cmp(v2.Gas) < 0 }

type vectorSummer struct {
	vectors []Vector
	by      func(v Vector) *big.Int
}

type VectorSum func(v Vector) *big.Int

func (v VectorSum) Sum(vectors []Vector) *big.Int {
	vs := vectorSummer{
		vectors: vectors,
		by:      v,
	}
	return Sum(vs)
}

func (v vectorSummer) Len() int           { return len(v.vectors) }
func (v vectorSummer) Sum(i int) *big.Int { return v.by(v.vectors[i]) }

func GasSum(v Vector) *big.Int { return v.Gas }

var etherInWei = new(big.Rat).SetInt(ethutil.String2Big("1000000000000000000"))

func GasPrice(bp, gl, ep *big.Int) *big.Int {
	BP := new(big.Rat).SetInt(bp)
	GL := new(big.Rat).SetInt(gl)
	EP := new(big.Rat).SetInt(ep)
	GP := new(big.Rat).Quo(BP, GL)
	GP = GP.Quo(GP, EP)

	return GP.Mul(GP, etherInWei).Num()
}
