package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Find(t *testing.T) {
	slice := []string{"1", "2", "3", "4", "5"}
	idx, found := Find(slice, "5")
	require.True(t, found)
	require.NotEqual(t, idx, -1)
	idx, found = Find(slice, "6")
	require.False(t, found)
	require.Equal(t, idx, -1)
}
func Test_AddToSet(t *testing.T) {
	slice := []string{"1", "2", "3", "4", "5"}
	idx, found := Find(slice, "6")
	require.False(t, found)
	require.Equal(t, idx, -1)
	err := Add(&slice, "6")
	require.NoError(t, err)
	idx, found = Find(slice, "6")
	require.True(t, found)
	require.NotEqual(t, idx, -1)
	err = Add(&slice, "6")
	require.Error(t, err)
}
func Test_RemoveFromSet(t *testing.T) {
	slice := []string{"1", "2", "3", "4", "5"}
	idx, found := Find(slice, "5")
	require.True(t, found)
	require.NotEqual(t, idx, -1)
	err := Remove(&slice, "5")
	require.NoError(t, err)
	idx, found = Find(slice, "5")
	require.False(t, found)
	require.Equal(t, idx, -1)
	err = Remove(&slice, "12")
	require.Error(t, err)
}
func Test_Update(t *testing.T) {
	slice := []string{"1", "2", "3", "4", "5"}
	idx, found := Find(slice, "5")
	require.True(t, found)
	require.NotEqual(t, idx, -1)
	err := Update(&slice, "5", "10")
	require.NoError(t, err)
	idx, found = Find(slice, "5")
	require.False(t, found)
	require.Equal(t, idx, -1)
	err = Update(&slice, "18", "10")
	require.Error(t, err)
}
