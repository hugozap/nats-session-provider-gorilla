package natsstore

import (
	"encoding/json"
	"net/http"

	"crypto/sha256"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/nats-io/nats.go"
)

type NATSSessionStore struct {
	js             nats.JetStreamContext
	keyPrefix      string
	defaultOptions *sessions.Options
	Codecs         []securecookie.Codec
}

func NewStore(js nats.JetStreamContext, keyPrefix string, keyPairs ...[]byte) (*NATSSessionStore, error) {
	_, err := js.KeyValue(keyPrefix + "_sessions")

	// If the key does not exist, create it
	if err != nil {
		_, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: keyPrefix + "_sessions",
		})
		if err != nil {
			return nil, err
		}
	}

	// from the keypairs create sha256 hashes in new array
	hashes := make([][]byte, len(keyPairs))
	for i, key := range keyPairs {
		hash := sha256.Sum256(key)
		hashes[i] = hash[:]
	}

	return &NATSSessionStore{
		js:        js,
		keyPrefix: keyPrefix,
		defaultOptions: &sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30, // 30 days
		},
		Codecs: securecookie.CodecsFromPairs(hashes...),
	}, nil
}

func (s *NATSSessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

func (s *NATSSessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	session.Options = s.defaultOptions
	session.IsNew = true

	// Check if the session ID is in the cookie
	if c, err := r.Cookie(name); err == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.Load(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}

	return session, nil
}

func (s *NATSSessionStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Delete if max-age is <=0 (it means "delete cookie immediately")
	if session.Options.MaxAge <= 0 {
		//TODO : move this to a separate delete function
		kv, err := s.js.KeyValue(s.keyPrefix + "_sessions")
		if err != nil {
			return err
		}
		err = kv.Delete(session.ID)
		if err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		session.ID = generateSessionID()
	}

	kv, err := s.js.KeyValue(s.keyPrefix + "_sessions")
	if err != nil {
		return err
	}

	values := make(map[string]string)
	for k, v := range session.Values {
		if key, ok := k.(string); ok {
			values[key] = v.(string)
		}
	}

	encoded, err := json.Marshal(values)

	if err != nil {
		return err
	}

	_, err = kv.Put(session.ID, encoded)
	if err != nil {
		return err
	}

	encodedValue, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encodedValue, session.Options))
	return nil
}

func (s *NATSSessionStore) Load(session *sessions.Session) error {

	kv, err := s.js.KeyValue(s.keyPrefix + "_sessions")
	if err != nil {
		return err
	}

	data, err := kv.Get(session.ID)
	if err != nil {
		return err
	}

	var values map[string]interface{}

	err = json.Unmarshal(data.Value(), &values)
	if err != nil {
		return err
	}

	for k, v := range values {
		session.Values[k] = v
	}

	return nil

}

func generateSessionID() string {

	return uuid.New().String()
}
