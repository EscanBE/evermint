package vm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AccessList(t *testing.T) {
	al2 := newAccessList2()
	testAccessList(t, al2)

	alGeth := newAccessList()
	testAccessList(t, alGeth)

	t.Run("Copy AccessList2", func(t *testing.T) {
		alSrc := newAccessList2()

		fillRandomDataToSrc := func() {
			for a := 10; a < 20; a++ {
				addr := GenerateAddress()
				alSrc.AddAddress(addr)

				if a%2 == 0 {
					for s := 0; s < a-8; s++ {
						alSrc.AddSlot(addr, GenerateHash())
					}
				}
			}
		}

		fillRandomDataToSrc()

		originalLen := len(alSrc.elements)
		originalSumLen := originalLen
		for _, v := range alSrc.elements {
			originalSumLen += len(v)
		}

		alCopy := alSrc.Copy()
		require.True(t, reflect.DeepEqual(alSrc.elements, alCopy.elements))

		fillRandomDataToSrc()
		require.False(t, reflect.DeepEqual(alSrc.elements, alCopy.elements))

		laterLen := len(alSrc.elements)
		laterSumLen := laterLen
		for _, v := range alSrc.elements {
			laterSumLen += len(v)
		}

		require.Less(t, originalSumLen, laterSumLen)
		require.Less(t, originalLen, laterLen)

		copyLen := len(alCopy.elements)
		copySumLen := copyLen
		for _, v := range alCopy.elements {
			copySumLen += len(v)
		}

		require.Equal(t, originalSumLen, copySumLen)
		require.Equal(t, originalLen, copyLen)
	})
}

func testAccessList(t *testing.T, al accessListI) {
	require.NotNil(t, al)

	addr1 := GenerateAddress()
	addr2 := GenerateAddress()

	require.False(t, al.ContainsAddress(addr1))
	require.False(t, al.ContainsAddress(addr2))

	al.AddAddress(addr1)
	require.True(t, al.ContainsAddress(addr1))
	require.False(t, al.ContainsAddress(addr2))

	al.AddAddress(addr2)
	require.True(t, al.ContainsAddress(addr2))

	slot1 := GenerateHash()
	slot2 := GenerateHash()

	existsAddr, existsSlot := al.Contains(addr1, slot1)
	require.True(t, existsAddr)
	require.False(t, existsSlot)

	addrChange, slotChange := al.AddSlot(addr1, slot1)
	require.False(t, addrChange)
	require.True(t, slotChange)

	addrChange, slotChange = al.AddSlot(addr1, slot1)
	require.False(t, addrChange)
	require.False(t, slotChange)

	existsAddr, existsSlot = al.Contains(addr1, slot1)
	require.True(t, existsAddr)
	require.True(t, existsSlot)

	existsAddr, existsSlot = al.Contains(addr1, slot2)
	require.True(t, existsAddr)
	require.False(t, existsSlot)

	existsAddr, existsSlot = al.Contains(GenerateAddress(), slot1)
	require.False(t, existsAddr)
	require.False(t, existsSlot)

	addr3 := GenerateAddress()
	require.False(t, al.ContainsAddress(addr3))
	require.True(t, al.AddAddress(addr3))
	require.True(t, al.ContainsAddress(addr3))
	require.False(t, al.AddAddress(addr3))

	require.NotPanics(t, func() {
		al.DeleteAddress(GenerateAddress())
	}, "should not panic even tho address is not in the access list")

	require.Panics(t, func() {
		al.DeleteSlot(GenerateAddress(), GenerateHash())
	}, "should panic if address is not in the access list")

	al.DeleteAddress(addr1)
	require.False(t, al.ContainsAddress(addr1))

	addrChange, slotChange = al.AddSlot(addr1, slot1)
	require.True(t, addrChange)
	require.True(t, slotChange)

	addrChange, slotChange = al.AddSlot(addr1, slot2)
	require.False(t, addrChange)
	require.True(t, slotChange)

	require.NotPanics(t, func() {
		al.DeleteSlot(addr1, GenerateHash())
	}, "should not panic even slot does not exists")

	require.NotPanics(t, func() {
		al.DeleteSlot(addr1, slot1)
	})
	if localAl, ok := al.(*AccessList2); ok {
		require.Len(t, localAl.elements[addr1], 1)
	}

	require.NotPanics(t, func() {
		al.DeleteSlot(addr1, slot2)
	})
	if localAl, ok := al.(*AccessList2); ok {
		require.Len(t, localAl.elements[addr1], 0)
		require.Nil(t, localAl.elements[addr1], "map should be erased")
	}
}
