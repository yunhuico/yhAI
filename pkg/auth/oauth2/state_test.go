package oauth2

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
)

type dummyOAuthDB struct {
	mu sync.Mutex

	states map[string]model.OAuth2State
}

func newDummyOAuthDB() *dummyOAuthDB {
	return &dummyOAuthDB{
		states: make(map[string]model.OAuth2State),
	}
}

func (d *dummyOAuthDB) UpdateOAuth2StateByID(ctx context.Context, state *model.OAuth2State, column ...string) (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, ok := d.states[state.ID]
	if !ok {
		err = fmt.Errorf("unknown ID %q", state.ID)
		return
	}

	d.states[state.ID] = *state

	return nil
}

func (d *dummyOAuthDB) InsertOAuth2State(ctx context.Context, state *model.OAuth2State) (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if state.ID == "" {
		state.ID, err = utils.NanoID()
		if err != nil {
			err = fmt.Errorf("generating nanoID: %w", err)
			return
		}
	}

	d.states[state.ID] = *state

	return nil
}

func (d *dummyOAuthDB) GetOAuth2StateByID(ctx context.Context, id string) (state model.OAuth2State, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	state, ok := d.states[id]
	if !ok {
		err = fmt.Errorf("unknown ID %q", state.ID)
		return
	}

	return
}

func TestDBStateStore(t *testing.T) {
	log.Init("go.test", log.DebugLevel)
	ctx := context.TODO()
	db := newDummyOAuthDB()
	stateStore := NewDBStateStore(db)
	require.NotNil(t, stateStore)

	t.Run("test add normal state success", func(t *testing.T) {
		stateKey, err := stateStore.AddState(ctx, NewState("credentialID", ""))
		require.NoError(t, err)

		state, err := stateStore.GetStateByKey(ctx, stateKey)
		require.NoError(t, err)
		require.Equal(t, "credentialID", state.CredentialID)
	})

	t.Run("test check a not exists state", func(t *testing.T) {
		_, err := stateStore.GetStateByKey(ctx, "not exists state key")
		require.Error(t, err)
	})

	t.Run("test callback state fail", func(t *testing.T) {
		state := NewCallbackState("credentialID")
		_, err := stateStore.AddState(ctx, state)
		require.NoError(t, err)
		go stateStore.ListenStateCompleted(ctx, state)
		err = db.UpdateOAuth2StateByID(ctx, &model.OAuth2State{
			ID:     state.ID,
			Status: model.OAuth2StatusFailed,
		})
		require.NoError(t, err)
		msg := <-state.ReceiveNotify()
		require.True(t, strings.HasPrefix(msg, "error:"))
	})
}
