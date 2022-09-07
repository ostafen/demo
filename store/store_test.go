package store_test

import (
	"demo/model"
	"demo/store"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func runTest(t *testing.T, testFunc func(store store.EventStore, t *testing.T)) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)

	store, err := store.Open(dir)
	require.NoError(t, err)

	testFunc(store, t)

	err = os.RemoveAll(dir)
	require.NoError(t, err)
}

func TestCreateAndGetAnswer(t *testing.T) {
	runTest(t, func(s store.EventStore, t *testing.T) {
		helloAnswer := &model.Answer{Key: "Hello", Value: "World!"}

		err := s.Create(helloAnswer)
		require.NoError(t, err)

		answ, err := s.GetAnswer("Hello")
		require.NoError(t, err)

		require.Equal(t, answ, helloAnswer)

		err = s.Create(helloAnswer)
		require.Equal(t, err, store.ErrAnswerExist)
	})
}

func TestUpdateAnswer(t *testing.T) {
	runTest(t, func(s store.EventStore, t *testing.T) {
		n := 100

		err := s.Create(&model.Answer{Key: "key", Value: "-1"})
		require.NoError(t, err)

		for i := 0; i < n; i++ {
			updateAnsw := &model.Answer{Key: "key", Value: strconv.Itoa(i)}

			err := s.Update(updateAnsw)
			require.NoError(t, err)

			answ, err := s.GetAnswer("key")
			require.NoError(t, err)

			require.Equal(t, answ, updateAnsw)
		}

		err = s.Update(&model.Answer{Key: "hello", Value: ""})
		require.Equal(t, err, store.ErrAnswerNotExist)
	})
}

func TestDeleteAnswer(t *testing.T) {
	runTest(t, func(s store.EventStore, t *testing.T) {
		n := 100

		for i := 0; i < n; i++ {
			err := s.Create(&model.Answer{Key: "key", Value: "key"})
			require.NoError(t, err)

			err = s.Delete("key")
			require.NoError(t, err)

			_, err = s.GetAnswer("key")
			require.Equal(t, err, store.ErrAnswerNotExist)
		}
	})
}

func randomEventType() model.EventType {
	switch rand.Intn(3) {
	case 0:
		return model.CreateEvent
	case 1:
		return model.UpdateEvent
	case 2:
		return model.DeleteEvent
	}
	panic("unknown event code")
}

func TestCreateUpdateAndDelete(t *testing.T) {
	runTest(t, func(s store.EventStore, t *testing.T) {
		n := 1000

		keyState := make(map[string]*string)

		for i := 0; i < n; i++ {
			key := strconv.Itoa(rand.Intn(10))

			evt := randomEventType()
			switch evt {
			case model.CreateEvent:
				value := strconv.Itoa(rand.Int())
				err := s.Create(&model.Answer{Key: key, Value: value})
				if keyState[key] == nil {
					require.NoError(t, err)
					keyState[key] = &value
				} else {
					require.Equal(t, err, store.ErrAnswerExist)
				}
			case model.UpdateEvent:
				value := strconv.Itoa(rand.Int())
				err := s.Update(&model.Answer{Key: key, Value: value})
				if keyState[key] != nil {
					require.NoError(t, err)
					keyState[key] = &value
				} else {
					require.Equal(t, err, store.ErrAnswerNotExist)
				}
			case model.DeleteEvent:
				err := s.Delete(key)
				if keyState[key] != nil {
					require.NoError(t, err)
					keyState[key] = nil
				} else {
					require.Equal(t, err, store.ErrAnswerNotExist)
				}
			}

			if keyState[key] != nil {
				a, err := s.GetAnswer(key)
				require.NoError(t, err)
				require.Equal(t, a, &model.Answer{Key: key, Value: *keyState[key]})
			}
		}
	})
}
