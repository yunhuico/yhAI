package oauth2

import (
	"context"
	"fmt"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
)

// TODO(sword): refactor this for supporting multiple UltraFox node deployment.
// UltraFox cli tool is next plan.

type State struct {
	model.OAuth2State

	notifyChan chan string
}

// NewCallbackState creates a new state for caller of created the state.
func NewCallbackState(credentialID string) *State {
	return &State{
		OAuth2State: model.OAuth2State{
			Mode:         model.OAuth2StateCliMode,
			CredentialID: credentialID,
		},
		notifyChan: make(chan string, 1),
	}
}

// NewState create a new state, no callback. It is used for the api.
func NewState(credentialID, redirectURL string) *State {
	return &State{
		OAuth2State: model.OAuth2State{
			Mode:         model.OAuth2StateNormalMode,
			CredentialID: credentialID,
			RedirectURL:  redirectURL,
		},
	}
}

// ListenStateCompleted send the credential connection to the channel.
func (d *DBStateStore) ListenStateCompleted(ctx context.Context, s *State) {
	utils.PanicIf(s.notifyChan == nil, "state notifyChan cannot be nil")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			latestState, err := d.db.GetOAuth2StateByID(ctx, s.ID)
			if err != nil {
				s.notifyChan <- fmt.Sprintf("error: %v", err)
				return
			}
			if latestState.Status == model.OAuth2StatusInit {
				continue
			}
			if latestState.Status == model.OAuth2StatusCompleted {
				s.notifyChan <- "ok"
			}
			if latestState.Status == model.OAuth2StatusFailed {
				s.notifyChan <- "error: check log"
			}
			return
		}
	}
}

// ReceiveNotify receives the credential connection from the channel.
func (s *State) ReceiveNotify() <-chan string {
	return s.notifyChan
}

// DBStateStore used to store state of oauth2
type DBStateStore struct {
	db oAuthDB
}

type oAuthDB interface {
	InsertOAuth2State(ctx context.Context, state *model.OAuth2State) (err error)
	GetOAuth2StateByID(ctx context.Context, id string) (state model.OAuth2State, err error)
	UpdateOAuth2StateByID(ctx context.Context, state *model.OAuth2State, column ...string) (err error)
}

// AddState add a request state to store
func (d *DBStateStore) AddState(ctx context.Context, state *State) (string, error) {
	err := d.db.InsertOAuth2State(ctx, &state.OAuth2State)
	if err != nil {
		err = fmt.Errorf("inserting OAuth2State: %w", err)
		return "", err
	}
	return state.ID, nil
}

func (d *DBStateStore) GetStateByKey(ctx context.Context, key string) (*State, error) {
	st, err := d.db.GetOAuth2StateByID(ctx, key)
	if err != nil {
		return nil, err
	}

	state := &State{
		OAuth2State: st,
	}

	return state, nil
}

func NewDBStateStore(db oAuthDB) *DBStateStore {
	return &DBStateStore{db: db}
}
