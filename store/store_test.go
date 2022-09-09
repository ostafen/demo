package store_test

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/ostafen/demo/model"
	"github.com/ostafen/demo/store"

	"github.com/stretchr/testify/require"
)

func runTest(t *testing.T, testFunc func(store store.EventStore, t *testing.T)) {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)

	store, err := store.Open(dir)
	require.NoError(t, err)

	testFunc(store, t)

	require.NoError(t, store.Close())
	require.NoError(t, os.RemoveAll(dir))
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

func TestGetHistory(t *testing.T) {
	runTest(t, func(s store.EventStore, t *testing.T) {
		n := 1000

		key := "key"
		err := s.Create(&model.Answer{Key: key, Value: "value"})
		require.NoError(t, err)

		evts := make([]*model.Event, 0)

		evts = append(evts, &model.Event{Event: model.CreateEvent, Data: &model.Answer{Key: key, Value: "value"}})

		for i := 0; i < n; i++ {
			switch randomEventType() {
			case model.CreateEvent:
				answ := &model.Answer{Key: key, Value: "value"}
				err := s.Create(answ)
				if err != store.ErrAnswerExist {
					require.NoError(t, err)
					evts = append(evts, &model.Event{Event: model.CreateEvent, Data: answ})
				}

			case model.UpdateEvent:
				answ := &model.Answer{Key: key, Value: "value"}
				err := s.Update(answ)
				if err != store.ErrAnswerNotExist {
					require.NoError(t, err)
					evts = append(evts, &model.Event{Event: model.UpdateEvent, Data: answ})
				}

			case model.DeleteEvent:
				answ := &model.Answer{Key: key, Value: ""}
				err := s.Delete(key)
				if err != store.ErrAnswerNotExist {
					require.NoError(t, err)
					evts = append(evts, &model.Event{Event: model.DeleteEvent, Data: answ})
				}
			}
		}

		it, err := s.GetHistory(key)
		require.NoError(t, err)

		i := 0
		for it.Next() {
			e, err := it.Value()
			require.NoError(t, err)
			require.Equal(t, e, evts[i])
			i++
		}
		require.Equal(t, i, len(evts))
		require.NoError(t, it.Close())
	})
}
