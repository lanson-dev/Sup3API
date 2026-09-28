package sup3

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	_ "github.com/lib/pq"
	"time"
)

// Store owns only sup3_ tables; no upstream migration or Ent schema is changed.
type Store struct{ DB *sql.DB }
type privateArtifact struct {
	SourceURL string
	Path      string
}
type record struct {
	Job                                        Job
	OwnerID, KeyID                             int64
	IdempotencyKey, RequestHash, ResolvedInput string
	Endpoints                                  []string
	Files                                      []privateArtifact
}

func encode(j *Job) ([]byte, error) {
	r := record{Job: *j, OwnerID: j.OwnerID, KeyID: j.KeyID, IdempotencyKey: j.IdempotencyKey, RequestHash: j.RequestHash, ResolvedInput: j.ResolvedInput}
	for _, s := range j.Steps {
		r.Endpoints = append(r.Endpoints, s.Endpoint)
	}
	for _, a := range j.Artifacts {
		r.Files = append(r.Files, privateArtifact{a.SourceURL, a.Path})
	}
	return json.Marshal(r)
}
func decode(b []byte) (*Job, error) {
	var r record
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	j := r.Job
	j.OwnerID = r.OwnerID
	j.KeyID = r.KeyID
	j.IdempotencyKey = r.IdempotencyKey
	j.RequestHash = r.RequestHash
	j.ResolvedInput = r.ResolvedInput
	for i := range j.Steps {
		if i < len(r.Endpoints) {
			j.Steps[i].Endpoint = r.Endpoints[i]
		}
	}
	for i := range j.Artifacts {
		if i < len(r.Files) {
			j.Artifacts[i].SourceURL = r.Files[i].SourceURL
			j.Artifacts[i].Path = r.Files[i].Path
		}
	}
	return &j, nil
}
func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func OpenStore(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	s := &Store{DB: db}
	if err = s.Init(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Init(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS sup3_jobs (
 id TEXT PRIMARY KEY, owner_id BIGINT NOT NULL, key_id BIGINT NOT NULL,
 idempotency_key TEXT NOT NULL, request_hash TEXT NOT NULL,
 active BOOLEAN NOT NULL, body JSONB NOT NULL,
 next_run TIMESTAMPTZ NOT NULL DEFAULT now(), lease_until TIMESTAMPTZ NOT NULL DEFAULT now(), lease_token TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(owner_id,key_id,idempotency_key));
 CREATE INDEX IF NOT EXISTS sup3_jobs_due ON sup3_jobs(next_run) WHERE active;
 CREATE INDEX IF NOT EXISTS sup3_jobs_owner ON sup3_jobs(owner_id,key_id,created_at DESC);`)
	return err
}
func (s *Store) Create(ctx context.Context, j *Job) (*Job, bool, error) {
	b, err := encode(j)
	if err != nil {
		return nil, false, err
	}
	result, err := s.DB.ExecContext(ctx, `INSERT INTO sup3_jobs(id,owner_id,key_id,idempotency_key,request_hash,active,body) VALUES($1,$2,$3,$4,$5,true,$6) ON CONFLICT(owner_id,key_id,idempotency_key) DO NOTHING`, j.ID, j.OwnerID, j.KeyID, j.IdempotencyKey, j.RequestHash, string(b))
	if err != nil {
		return nil, false, err
	}
	n, _ := result.RowsAffected()
	if n == 1 {
		return j, true, nil
	}
	var old []byte
	err = s.DB.QueryRowContext(ctx, `SELECT body FROM sup3_jobs WHERE owner_id=$1 AND key_id=$2 AND idempotency_key=$3`, j.OwnerID, j.KeyID, j.IdempotencyKey).Scan(&old)
	if err != nil {
		return nil, false, err
	}
	prev, err := decode(old)
	if err != nil {
		return nil, false, err
	}
	if prev.RequestHash != j.RequestHash {
		return nil, false, &APIError{Code: "idempotency_conflict", Message: "Idempotency-Key was already used with another request", HTTPStatus: 409}
	}
	return prev, false, nil
}
func (s *Store) Get(ctx context.Context, id string, owner, key int64) (*Job, error) {
	var b []byte
	err := s.DB.QueryRowContext(ctx, `SELECT body FROM sup3_jobs WHERE id=$1 AND owner_id=$2 AND key_id=$3`, id, owner, key).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &APIError{Code: "not_found", Message: "job not found", HTTPStatus: 404}
	}
	if err != nil {
		return nil, err
	}
	return decode(b)
}
func (s *Store) List(ctx context.Context, owner, key int64) ([]*Job, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT body FROM sup3_jobs WHERE owner_id=$1 AND key_id=$2 ORDER BY created_at DESC LIMIT 100`, owner, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Job{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		j, e := decode(b)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// Claims use fencing tokens and a durable lease. A crashed POST is never retried.
func (s *Store) Claim(ctx context.Context, id string) (*Job, string, error) {
	token := randomID()
	var b []byte
	err := s.DB.QueryRowContext(ctx, `UPDATE sup3_jobs SET lease_token=$1,lease_until=now()+interval '5 minutes'
 WHERE id=(SELECT id FROM sup3_jobs WHERE ($2='' AND active AND next_run<=now() OR $2<>'' AND id=$2) AND lease_until<=now() ORDER BY next_run FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING body`, token, id).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	j, err := decode(b)
	return j, token, err
}
func (s *Store) Save(ctx context.Context, j *Job, token string, release bool) error {
	j.UpdatedAt = time.Now().UTC()
	b, err := encode(j)
	if err != nil {
		return err
	}
	active := !j.Terminal() || (j.Status == "succeeded" && j.DeliveryStatus == "pending")
	result, err := s.DB.ExecContext(ctx, `UPDATE sup3_jobs SET body=$1,active=$2,next_run=now()+interval '5 seconds',lease_until=CASE WHEN $3 THEN now() ELSE lease_until END WHERE id=$4 AND lease_token=$5 AND lease_until>now()`, string(b), active, release, j.ID, token)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("sup3 job lease lost")
	}
	return nil
}
