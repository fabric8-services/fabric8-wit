package id_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/fabric8-services/fabric8-wit/id"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestSliceDiff(t *testing.T) {
	a := id.Slice{
		// shared with b
		uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94"),
		uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8"),
		// unique
		uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809"),
		uuid.FromStringOrNil("0b5159b5-c21b-40c2-af90-f020d71a8e94"),
	}
	b := id.Slice{
		// shared with a
		uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94"),
		uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8"),
		// unique to b
		uuid.FromStringOrNil("03a9a225-e7b0-4229-b698-716308f2136a"),
		uuid.FromStringOrNil("1db1c165-2360-4efc-89b4-e4d3d4988091"),
		uuid.FromStringOrNil("2fc09162-bf2f-4a1c-b622-323d8495ac58"),
	}
	t.Run("equality", func(t *testing.T) {
		require.Equal(t, id.Slice{}, a.Diff(a))
	})
	diffs := []id.Slice{a.Diff(b),
		b.Diff(a),
	}
	for _, d := range diffs {
		t.Run("difference", func(t *testing.T) {
			toBeFound := map[uuid.UUID]struct{}{
				// unique to a
				uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809"): {},
				uuid.FromStringOrNil("0b5159b5-c21b-40c2-af90-f020d71a8e94"): {},
				// unique to b
				uuid.FromStringOrNil("03a9a225-e7b0-4229-b698-716308f2136a"): {},
				uuid.FromStringOrNil("1db1c165-2360-4efc-89b4-e4d3d4988091"): {},
				uuid.FromStringOrNil("2fc09162-bf2f-4a1c-b622-323d8495ac58"): {},
			}
			// then
			for _, i := range d {
				_, ok := toBeFound[i]
				require.True(t, ok, "failed to find %s in expected difference: %s", i, toBeFound)
				delete(toBeFound, i)
			}
			require.Empty(t, toBeFound, "found not all IDs in difference: %+v", toBeFound)
		})
	}
}

func TestSliceSub(t *testing.T) {
	a := id.Slice{
		// shared with b
		uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94"),
		uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8"),
		// unique
		uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809"),
		uuid.FromStringOrNil("0b5159b5-c21b-40c2-af90-f020d71a8e94"),
	}
	b := id.Slice{
		// shared with a
		uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94"),
		uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8"),
		// unique to b
		uuid.FromStringOrNil("03a9a225-e7b0-4229-b698-716308f2136a"),
		uuid.FromStringOrNil("1db1c165-2360-4efc-89b4-e4d3d4988091"),
		uuid.FromStringOrNil("2fc09162-bf2f-4a1c-b622-323d8495ac58"),
	}
	t.Run("equality", func(t *testing.T) {
		require.Empty(t, a.Sub(a))
	})
	t.Run("unchanged when subtracting nothing", func(t *testing.T) {
		require.Equal(t, a, a.Sub(id.Slice{}))
	})
	t.Run("unchanged when subtracting non existent ID", func(t *testing.T) {
		require.Equal(t, a, a.Sub(id.Slice{uuid.FromStringOrNil("29976200-df70-49d9-9fb1-789de14c5cec")}))
	})
	t.Run("a-b", func(t *testing.T) {
		toBeFound := map[uuid.UUID]struct{}{
			// unique to a
			uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809"): {},
			uuid.FromStringOrNil("0b5159b5-c21b-40c2-af90-f020d71a8e94"): {},
		}
		// when
		res := a.Sub(b)
		// then
		for _, i := range res {
			_, ok := toBeFound[i]
			require.True(t, ok, "failed to find %s in expected subtraction result: %s", i, toBeFound)
			delete(toBeFound, i)
		}
		require.Empty(t, toBeFound, "found not all IDs in subtraction result: %+v", toBeFound)
	})
	t.Run("b-a", func(t *testing.T) {
		toBeFound := map[uuid.UUID]struct{}{
			// unique to b
			uuid.FromStringOrNil("03a9a225-e7b0-4229-b698-716308f2136a"): {},
			uuid.FromStringOrNil("1db1c165-2360-4efc-89b4-e4d3d4988091"): {},
			uuid.FromStringOrNil("2fc09162-bf2f-4a1c-b622-323d8495ac58"): {},
		}
		// when
		res := b.Sub(a)
		// then
		for _, i := range res {
			_, ok := toBeFound[i]
			require.True(t, ok, "failed to find %s in expected subtraction result: %s", i, toBeFound)
			delete(toBeFound, i)
		}
		require.Empty(t, toBeFound, "found not all IDs in subtraction result: %+v", toBeFound)
	})
}

func TestSliceAdd(t *testing.T) {
	a1 := uuid.FromStringOrNil("cb4364bf-893c-4ac2-a35f-4ef74c212e88")
	b1 := uuid.FromStringOrNil("d085a858-6b98-44ce-8777-b36d927b07dc")
	t.Run("a+b", func(t *testing.T) {
		a := id.Slice{a1}
		b := id.Slice{b1}
		a.Add(b)
		require.Equal(t, id.Slice{a1, b1}, a)
	})
	t.Run("b+a", func(t *testing.T) {
		a := id.Slice{a1}
		b := id.Slice{b1}
		b.Add(a)
		require.Equal(t, id.Slice{b1, a1}, b)
	})
}

func TestSliceToString(t *testing.T) {
	// given
	a := uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94")
	b := uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8")
	c := uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809")
	s := id.Slice{a, b, c}
	// when
	res := s.ToString("; ", func(ID uuid.UUID) string { return fmt.Sprintf("(%s)", ID) })
	// then
	require.Equal(t, fmt.Sprintf("(%s); (%s); (%s)", a, b, c), res)
}

func TestSliceString(t *testing.T) {
	// given
	a := uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94")
	b := uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8")
	c := uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809")
	s := id.Slice{a, b, c}
	// when
	res := s.String()
	// then
	require.Equal(t, fmt.Sprintf("%s, %s, %s", a, b, c), res)
}

func TestSliceSort(t *testing.T) {
	// given
	a := uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94")
	b := uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8")
	c := uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809")
	s := id.Slice{a, b, c}
	// when
	sort.Sort(s)
	// then
	require.Equal(t, id.Slice{c, b, a}, s)
}
