package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"

	"github.com/stretchr/testify/require"
)

func TestWebhookTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		t       WebhookTime
		want    []byte
		wantErr bool
	}{
		{
			name:    "5 digit year",
			t:       WebhookTime(time.Unix(999999999999999, 0)), // 33658/9/27 09:46:39 GMT+0800
			want:    nil,
			wantErr: true,
		},
		{
			name:    "year digit below 0",
			t:       WebhookTime(time.Unix(-999999999999999, 0)), // a really ancient time
			want:    nil,
			wantErr: true,
		},
		{
			name:    "good",
			t:       WebhookTime(time.Unix(1667980001, 0).UTC()), // specify a timezone to make test repeatable anywhere
			want:    []byte(`"2022-11-09 07:46:41 +0000"`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.t.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestWebhookTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantUnix int64
		wantErr  bool
	}{
		{
			name:     "null",
			data:     []byte("null"),
			wantUnix: time.Time{}.Unix(),
			wantErr:  false,
		},
		{
			name:     "Jihulab",
			data:     []byte(`"2022-11-09 14:11:26 +0800"`),
			wantUnix: 1667974286,
			wantErr:  false,
		},
		{
			name:     "Gitlab",
			data:     []byte(`"2022-11-09 07:24:10 UTC"`),
			wantUnix: 1667978650,
			wantErr:  false,
		},
		{
			name:     "unexpected layout",
			data:     []byte(`"2006-01-02T15:04:05Z07:00"`),
			wantUnix: 0,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got WebhookTime
			err := json.Unmarshal(tt.data, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			gotTime := time.Time(got)
			if gotTime.Unix() != tt.wantUnix {
				t.Errorf("got time %s, unix %d, want unix %d", gotTime, gotTime.Unix(), tt.wantUnix)
			}
		})
	}
}

// TestIssueTrigger_Run is a test against https://jihulab.com/ultrafox/ultrafox/-/issues/353#note_1514647
func TestIssueTrigger_Run(t *testing.T) {
	const body = `{
  "object_kind": "issue",
  "event_type": "issue",
  "user": {
    "id": 4391,
    "name": "Sword",
    "username": "sword",
    "avatar_url": "https://jihulab.com/uploads/-/system/user/avatar/4391/avatar.png",
    "email": "[REDACTED]"
  },
  "project": {
    "id": 23509,
    "name": "evenyone-is-owner",
    "description": "",
    "web_url": "https://jihulab.com/sword/evenyone-is-owner",
    "avatar_url": null,
    "git_ssh_url": "git@jihulab.com:sword/evenyone-is-owner.git",
    "git_http_url": "https://jihulab.com/sword/evenyone-is-owner.git",
    "namespace": "Sword",
    "visibility_level": 20,
    "path_with_namespace": "sword/evenyone-is-owner",
    "default_branch": "main",
    "ci_config_path": "",
    "homepage": "https://jihulab.com/sword/evenyone-is-owner",
    "url": "git@jihulab.com:sword/evenyone-is-owner.git",
    "ssh_url": "git@jihulab.com:sword/evenyone-is-owner.git",
    "http_url": "https://jihulab.com/sword/evenyone-is-owner.git"
  },
  "object_attributes": {
    "author_id": 4391,
    "closed_at": null,
    "confidential": false,
    "created_at": "2022-11-09 16:58:23 +0800",
    "description": "",
    "discussion_locked": null,
    "due_date": null,
    "id": 254609,
    "iid": 68,
    "last_edited_at": null,
    "last_edited_by_id": null,
    "milestone_id": null,
    "moved_to_id": null,
    "duplicated_to_id": null,
    "project_id": 23509,
    "relative_position": null,
    "state_id": 1,
    "time_estimate": 0,
    "title": "hixix",
    "updated_at": "2022-11-09 16:58:23 +0800",
    "updated_by_id": null,
    "weight": null,
    "url": "https://jihulab.com/sword/evenyone-is-owner/-/issues/68",
    "total_time_spent": 0,
    "time_change": 0,
    "human_total_time_spent": null,
    "human_time_change": null,
    "human_time_estimate": null,
    "assignee_ids": [

    ],
    "assignee_id": null,
    "labels": [
      {
        "id": 17201,
        "title": "123",
        "color": "#6699cc",
        "project_id": 23509,
        "created_at": "2022-05-25 11:31:25 +0800",
        "updated_at": "2022-05-25 11:31:25 +0800",
        "template": false,
        "description": null,
        "type": "ProjectLabel",
        "group_id": null
      },
      {
        "id": 53617,
        "title": "asdasdasd",
        "color": "#6699cc",
        "project_id": 23509,
        "created_at": "2022-11-08 17:56:12 +0800",
        "updated_at": "2022-11-08 17:56:12 +0800",
        "template": false,
        "description": null,
        "type": "ProjectLabel",
        "group_id": null
      }
    ],
    "state": "opened",
    "severity": "unknown",
    "action": "open"
  },
  "labels": [
    {
      "id": 17201,
      "title": "123",
      "color": "#6699cc",
      "project_id": 23509,
      "created_at": "2022-05-25 11:31:25 +0800",
      "updated_at": "2022-05-25 11:31:25 +0800",
      "template": false,
      "description": null,
      "type": "ProjectLabel",
      "group_id": null
    },
    {
      "id": 53617,
      "title": "asdasdasd",
      "color": "#6699cc",
      "project_id": 23509,
      "created_at": "2022-11-08 17:56:12 +0800",
      "updated_at": "2022-11-08 17:56:12 +0800",
      "template": false,
      "description": null,
      "type": "ProjectLabel",
      "group_id": null
    }
  ],
  "changes": {
    "author_id": {
      "previous": null,
      "current": 4391
    },
    "created_at": {
      "previous": null,
      "current": "2022-11-09 16:58:23 +0800"
    },
    "description": {
      "previous": null,
      "current": ""
    },
    "id": {
      "previous": null,
      "current": 254609
    },
    "iid": {
      "previous": null,
      "current": 68
    },
    "project_id": {
      "previous": null,
      "current": 23509
    },
    "title": {
      "previous": null,
      "current": "hixix"
    },
    "updated_at": {
      "previous": null,
      "current": "2022-11-09 16:58:23 +0800"
    }
  },
  "repository": {
    "name": "evenyone-is-owner",
    "url": "git@jihulab.com:sword/evenyone-is-owner.git",
    "description": "",
    "homepage": "https://jihulab.com/sword/evenyone-is-owner"
  }
}`

	assert := require.New(t)
	var trigger = IssueTrigger{
		Trigger{
			Header: nil,
			Query:  nil,
			Body:   []byte(body),
		},
	}

	got, err := trigger.Run(nil)
	assert.NoError(err)

	event, ok := got.(*IssueEvent)
	assert.True(ok)
	assert.Equal(int64(1667984303), event.ObjectAttributes.CreatedAt.Time().Unix())
}

// TestNoteTrigger_Run is a test against https://jihulab.com/ultrafox/ultrafox/-/issues/353#note_1514647
func TestNoteTrigger_Run(t *testing.T) {
	const body = `{
  "object_kind": "note",
  "event_type": "note",
  "user": {
    "id": 2691,
    "name": "LI Zhennan",
    "username": "nanmu42",
    "avatar_url": "https://jihulab.com/uploads/-/system/user/avatar/2691/avatar.png",
    "email": "[REDACTED]"
  },
  "project_id": 32885,
  "project": {
    "id": 32885,
    "name": "e2e test",
    "description": "",
    "web_url": "https://jihulab.com/ultrafox-dev/e2e-test",
    "avatar_url": null,
    "git_ssh_url": "git@jihulab.com:ultrafox-dev/e2e-test.git",
    "git_http_url": "https://jihulab.com/ultrafox-dev/e2e-test.git",
    "namespace": "Ultrafox Dev",
    "visibility_level": 20,
    "path_with_namespace": "ultrafox-dev/e2e-test",
    "default_branch": "main",
    "ci_config_path": "",
    "homepage": "https://jihulab.com/ultrafox-dev/e2e-test",
    "url": "git@jihulab.com:ultrafox-dev/e2e-test.git",
    "ssh_url": "git@jihulab.com:ultrafox-dev/e2e-test.git",
    "http_url": "https://jihulab.com/ultrafox-dev/e2e-test.git"
  },
  "object_attributes": {
    "attachment": null,
    "author_id": 24654,
    "change_position": null,
    "commit_id": null,
    "created_at": "2022-11-09 13:53:03 +0800",
    "discussion_id": "f9ade54c622b34366ba86e10748c1a4f7df8de4e",
    "id": 1514466,
    "line_code": null,
    "note": "Hello from Ultrafox Azure!",
    "noteable_id": 253125,
    "noteable_type": "Issue",
    "original_position": null,
    "position": null,
    "project_id": 32885,
    "resolved_at": null,
    "resolved_by_id": null,
    "resolved_by_push": null,
    "st_diff": null,
    "system": false,
    "type": null,
    "updated_at": "2022-11-09 13:53:03 +0800",
    "updated_by_id": null,
    "description": "Hello from Ultrafox Azure!",
    "url": "https://jihulab.com/ultrafox-dev/e2e-test/-/issues/940#note_1514466"
  },
  "repository": {
    "name": "e2e test",
    "url": "git@jihulab.com:ultrafox-dev/e2e-test.git",
    "description": "",
    "homepage": "https://jihulab.com/ultrafox-dev/e2e-test"
  },
  "issue": {
    "author_id": 2691,
    "closed_at": null,
    "confidential": false,
    "created_at": "2022-11-07 17:16:55 +0800",
    "description": "",
    "discussion_locked": null,
    "due_date": null,
    "id": 253125,
    "iid": 940,
    "last_edited_at": null,
    "last_edited_by_id": null,
    "milestone_id": null,
    "moved_to_id": null,
    "duplicated_to_id": null,
    "project_id": 32885,
    "relative_position": 780273,
    "state_id": 1,
    "time_estimate": 0,
    "title": "cron messages",
    "updated_at": "2022-11-09 13:53:03 +0800",
    "updated_by_id": null,
    "weight": null,
    "url": "https://jihulab.com/ultrafox-dev/e2e-test/-/issues/940",
    "total_time_spent": 0,
    "time_change": 0,
    "human_total_time_spent": null,
    "human_time_change": null,
    "human_time_estimate": null,
    "assignee_ids": [

    ],
    "assignee_id": null,
    "labels": [

    ],
    "state": "opened",
    "severity": "unknown"
  }
}`

	assert := require.New(t)
	var trigger = NoteTrigger{
		Trigger{
			Header: nil,
			Query:  nil,
			Body:   []byte(body),
		},
	}

	got, err := trigger.Run(nil)
	assert.NoError(err)

	event, ok := got.(*NoteEvent)
	assert.True(ok)
	assert.Equal(int64(1667973183), event.ObjectAttributes.CreatedAt.Time().Unix())
}

// TestMergeRequestTrigger_Run is a test against https://jihulab.com/ultrafox/ultrafox/-/issues/353#note_1514647
func TestMergeRequestTrigger_Run(t *testing.T) {
	const body = `{
  "object_kind": "merge_request",
  "event_type": "merge_request",
  "user": {
    "id": 2691,
    "name": "LI Zhennan",
    "username": "nanmu42",
    "avatar_url": "https://jihulab.com/uploads/-/system/user/avatar/2691/avatar.png",
    "email": "[REDACTED]"
  },
  "project": {
    "id": 30385,
    "name": "foo",
    "description": "A repo for fooling around.",
    "web_url": "https://jihulab.com/nanmu42/foo",
    "avatar_url": null,
    "git_ssh_url": "git@jihulab.com:nanmu42/foo.git",
    "git_http_url": "https://jihulab.com/nanmu42/foo.git",
    "namespace": "LI Zhennan",
    "visibility_level": 20,
    "path_with_namespace": "nanmu42/foo",
    "default_branch": "main",
    "ci_config_path": "",
    "homepage": "https://jihulab.com/nanmu42/foo",
    "url": "git@jihulab.com:nanmu42/foo.git",
    "ssh_url": "git@jihulab.com:nanmu42/foo.git",
    "http_url": "https://jihulab.com/nanmu42/foo.git"
  },
  "object_attributes": {
    "assignee_id": null,
    "author_id": 2691,
    "created_at": "2022-11-09 16:29:34 +0800",
    "description": "",
    "head_pipeline_id": null,
    "id": 155809,
    "iid": 1,
    "last_edited_at": null,
    "last_edited_by_id": null,
    "merge_commit_sha": null,
    "merge_error": null,
    "merge_params": {
      "force_remove_source_branch": "1"
    },
    "merge_status": "can_be_merged",
    "merge_user_id": null,
    "merge_when_pipeline_succeeds": false,
    "milestone_id": null,
    "source_branch": "feat/hello-world",
    "source_project_id": 30385,
    "state_id": 1,
    "target_branch": "main",
    "target_project_id": 30385,
    "time_estimate": 0,
    "title": "Feat/hello world",
    "updated_at": "2022-11-09 16:29:34 +0800",
    "updated_by_id": null,
    "url": "https://jihulab.com/nanmu42/foo/-/merge_requests/1",
    "source": {
      "id": 30385,
      "name": "foo",
      "description": "A repo for fooling around.",
      "web_url": "https://jihulab.com/nanmu42/foo",
      "avatar_url": null,
      "git_ssh_url": "git@jihulab.com:nanmu42/foo.git",
      "git_http_url": "https://jihulab.com/nanmu42/foo.git",
      "namespace": "LI Zhennan",
      "visibility_level": 20,
      "path_with_namespace": "nanmu42/foo",
      "default_branch": "main",
      "ci_config_path": "",
      "homepage": "https://jihulab.com/nanmu42/foo",
      "url": "git@jihulab.com:nanmu42/foo.git",
      "ssh_url": "git@jihulab.com:nanmu42/foo.git",
      "http_url": "https://jihulab.com/nanmu42/foo.git"
    },
    "target": {
      "id": 30385,
      "name": "foo",
      "description": "A repo for fooling around.",
      "web_url": "https://jihulab.com/nanmu42/foo",
      "avatar_url": null,
      "git_ssh_url": "git@jihulab.com:nanmu42/foo.git",
      "git_http_url": "https://jihulab.com/nanmu42/foo.git",
      "namespace": "LI Zhennan",
      "visibility_level": 20,
      "path_with_namespace": "nanmu42/foo",
      "default_branch": "main",
      "ci_config_path": "",
      "homepage": "https://jihulab.com/nanmu42/foo",
      "url": "git@jihulab.com:nanmu42/foo.git",
      "ssh_url": "git@jihulab.com:nanmu42/foo.git",
      "http_url": "https://jihulab.com/nanmu42/foo.git"
    },
    "last_commit": {
      "id": "e17abdda7b21186c0f2f8d1ed34b7a3904b03f7b",
      "message": "Hello Again!\n",
      "title": "Hello Again!",
      "timestamp": "2022-09-05T11:25:21+08:00",
      "url": "https://jihulab.com/nanmu42/foo/-/commit/e17abdda7b21186c0f2f8d1ed34b7a3904b03f7b",
      "author": {
        "name": "nanmu42",
        "email": "[REDACTED]"
      }
    },
    "work_in_progress": false,
    "total_time_spent": 0,
    "time_change": 0,
    "human_total_time_spent": null,
    "human_time_change": null,
    "human_time_estimate": null,
    "assignee_ids": [

    ],
    "reviewer_ids": [

    ],
    "labels": [

    ],
    "state": "opened",
    "blocking_discussions_resolved": true,
    "first_contribution": false,
    "detailed_merge_status": "mergeable"
  },
  "labels": [

  ],
  "changes": {
  },
  "repository": {
    "name": "foo",
    "url": "git@jihulab.com:nanmu42/foo.git",
    "description": "A repo for fooling around.",
    "homepage": "https://jihulab.com/nanmu42/foo"
  }
}`

	assert := require.New(t)
	var trigger = MergeRequestTrigger{
		Trigger{
			Header: nil,
			Query:  nil,
			Body:   []byte(body),
		},
	}

	got, err := trigger.Run(nil)
	assert.NoError(err)

	event, ok := got.(*MergeRequestEvent)
	assert.True(ok)
	assert.Equal(int64(1667982574), event.ObjectAttributes.CreatedAt.Time().Unix())
}

// TestReleaseTrigger_Run is a test against https://jihulab.com/ultrafox/ultrafox/-/issues/353#note_1514647
func TestReleaseTrigger_Run(t *testing.T) {
	// see? the commit timestamp uses RFC3339 but created_at does not
	const body = `{
  "id": 19133,
  "created_at": "2022-11-09 16:32:30 +0800",
  "description": "",
  "name": "Woo!",
  "released_at": "2022-11-09 16:32:30 +0800",
  "tag": "v1",
  "object_kind": "release",
  "project": {
    "id": 30385,
    "name": "foo",
    "description": "A repo for fooling around.",
    "web_url": "https://jihulab.com/nanmu42/foo",
    "avatar_url": null,
    "git_ssh_url": "git@jihulab.com:nanmu42/foo.git",
    "git_http_url": "https://jihulab.com/nanmu42/foo.git",
    "namespace": "LI Zhennan",
    "visibility_level": 20,
    "path_with_namespace": "nanmu42/foo",
    "default_branch": "main",
    "ci_config_path": "",
    "homepage": "https://jihulab.com/nanmu42/foo",
    "url": "git@jihulab.com:nanmu42/foo.git",
    "ssh_url": "git@jihulab.com:nanmu42/foo.git",
    "http_url": "https://jihulab.com/nanmu42/foo.git"
  },
  "url": "https://jihulab.com/nanmu42/foo/-/releases/v1",
  "action": "create",
  "assets": {
    "count": 4,
    "links": [

    ],
    "sources": [
      {
        "format": "zip",
        "url": "https://jihulab.com/nanmu42/foo/-/archive/v1/foo-v1.zip"
      },
      {
        "format": "tar.gz",
        "url": "https://jihulab.com/nanmu42/foo/-/archive/v1/foo-v1.tar.gz"
      },
      {
        "format": "tar.bz2",
        "url": "https://jihulab.com/nanmu42/foo/-/archive/v1/foo-v1.tar.bz2"
      },
      {
        "format": "tar",
        "url": "https://jihulab.com/nanmu42/foo/-/archive/v1/foo-v1.tar"
      }
    ]
  },
  "commit": {
    "id": "29859bcc73cdab4db2b70ed681077a5885f80134",
    "message": "Initial commit",
    "title": "Initial commit",
    "timestamp": "2022-06-22T13:47:19+08:00",
    "url": "https://jihulab.com/nanmu42/foo/-/commit/29859bcc73cdab4db2b70ed681077a5885f80134",
    "author": {
      "name": "LI Zhennan",
      "email": "[REDACTED]"
    }
  }
}`

	assert := require.New(t)
	var trigger = ReleaseTrigger{
		Trigger{
			Header: nil,
			Query:  nil,
			Body:   []byte(body),
		},
	}

	got, err := trigger.Run(nil)
	assert.NoError(err)

	event, ok := got.(*ReleaseEvent)
	assert.True(ok)
	assert.Equal(int64(1667982750), event.CreatedAt.Time().Unix())
	assert.Equal(int64(1655876839), event.Commit.Timestamp.Unix())
}

type mockAuthorizer struct {
	URL string
}

func (m mockAuthorizer) GetAccessToken(ctx context.Context) (string, error) {
	return "hello", nil
}

func (m mockAuthorizer) DecodeMeta(meta interface{}) (err error) {
	marshaled := fmt.Sprintf(`{"server": "%s"}`, m.URL)
	err = json.Unmarshal([]byte(marshaled), &meta)
	if err != nil {
		err = fmt.Errorf("unmarshaling into meta: %w", err)
		return
	}
	return
}

func (m mockAuthorizer) DecodeTokenMetaData(ctx context.Context, meta interface{}) error {
	return errors.New("not implemented")
}

func (m mockAuthorizer) CredentialType() model.CredentialType {
	return model.CredentialTypeAccessToken
}

type mockWebhookContext struct {
	log.Logger
	authorizer auth.Authorizer
	config     *TriggerConfig
}

func (m mockWebhookContext) Context() context.Context {
	return context.TODO()
}

func (m mockWebhookContext) GetConfigObject() any {
	return m.config
}

func (m mockWebhookContext) GetAuthorizer() auth.Authorizer {
	return m.authorizer
}

func (m mockWebhookContext) GetTriggerData() map[string]any {
	return map[string]any{
		"webhookID": 1,
	}
}

func (m mockWebhookContext) GetWebhookURL() string {
	return ""
}

func (m mockWebhookContext) SetTriggerQueryID(queryID string) {
	return
}

func (m mockWebhookContext) GetPassportVendorLookup() map[model.PassportVendorName]model.PassportVendor {
	return make(map[model.PassportVendorName]model.PassportVendor)
}

func TestTrigger_Exists_Unauthorized(t *testing.T) {
	var (
		assert = require.New(t)
		err    error
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}))
	defer server.Close()

	ctx := mockWebhookContext{
		Logger:     nil,
		authorizer: mockAuthorizer{URL: server.URL},
		config:     &TriggerConfig{},
	}

	var g Trigger
	_, err = g.Exists(ctx)
	assert.True(errors.Is(err, trigger.ErrTokenUnauthorized))

	err = g.Delete(ctx)
	assert.True(errors.Is(err, trigger.ErrTokenUnauthorized))
}
