package sup3

import (
	"context"
	"database/sql"
	"errors"
)

type nativeScope struct {
	Owner, Key           int64
	Provider, Credential string
}

type nativeCall struct {
	ID, Hash, TaskID, Endpoint string
	Status                     int
	Body                       []byte
}

func (s *Store) reserveNative(ctx context.Context, scope nativeScope, idem, hash, endpoint string) (nativeCall, bool, error) {
	c := nativeCall{ID: "native_" + randomID(), Hash: hash, Endpoint: endpoint}
	if idem == "" {
		idem = c.ID
	}
	result, err := s.DB.ExecContext(ctx, `INSERT INTO sup3_native_calls(id,owner_id,key_id,provider,credential,idempotency_key,request_hash,endpoint) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(owner_id,key_id,provider,credential,idempotency_key) DO NOTHING`, c.ID, scope.Owner, scope.Key, scope.Provider, scope.Credential, idem, hash, endpoint)
	if err != nil {
		return c, false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 1 {
		return c, n == 1, err
	}
	err = s.DB.QueryRowContext(ctx, `SELECT id,request_hash,task_id,endpoint,status,response FROM sup3_native_calls WHERE owner_id=$1 AND key_id=$2 AND provider=$3 AND credential=$4 AND idempotency_key=$5`, scope.Owner, scope.Key, scope.Provider, scope.Credential, idem).Scan(&c.ID, &c.Hash, &c.TaskID, &c.Endpoint, &c.Status, &c.Body)
	return c, false, err
}

func (s *Store) finishNative(ctx context.Context, id, task string, status int, body []byte) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE sup3_native_calls SET task_id=$2,status=$3,response=$4 WHERE id=$1`, id, task, status, body)
	return err
}

func (s *Store) nativeTask(ctx context.Context, scope nativeScope, task string) (string, error) {
	var endpoint string
	err := s.DB.QueryRowContext(ctx, `SELECT endpoint FROM sup3_native_calls WHERE owner_id=$1 AND key_id=$2 AND provider=$3 AND credential=$4 AND task_id=$5 AND task_id<>'' LIMIT 1`, scope.Owner, scope.Key, scope.Provider, scope.Credential, task).Scan(&endpoint)
	if errors.Is(err, sql.ErrNoRows) {
		return "", &APIError{Code: "native_task_not_found", Message: "task must have been created through this native gateway with the same API key", HTTPStatus: 404}
	}
	return endpoint, err
}
