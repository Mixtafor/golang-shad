//go:build !solution

package retryupdate

import (
	"errors"

	"github.com/gofrs/uuid"
	"gitlab.com/slon/shad-go/retryupdate/kvapi"
)

func UpdateValue(c kvapi.Client, key string, updateFn func(oldValue *string) (newValue string, err error)) error {

	var conflictErr *kvapi.ConflictError
	var authErr *kvapi.AuthError
OUTER_FOR:
	for {
		getResp, get_err := c.Get(&kvapi.GetRequest{Key: key})
		var newValue string

		switch {
		case errors.Is(get_err, kvapi.ErrKeyNotFound):
			updnewValue, err := updateFn(nil)
			newValue = updnewValue
			if err != nil {
				return err
			}
		case get_err == nil:
			updnewValue, err := updateFn(&getResp.Value)
			newValue = updnewValue
			if err != nil {
				return err
			}
		case errors.As(get_err, &authErr):
			return &kvapi.APIError{Method: "get", Err: authErr}
		default:
			continue
		}

		var old_uuid uuid.UUID
		if errors.Is(get_err, kvapi.ErrKeyNotFound) {
			old_uuid = uuid.UUID{}
		} else {
			old_uuid = getResp.Version
		}
		new_uuid := uuid.Must(uuid.NewV4())

		for {
			_, err := c.Set(&kvapi.SetRequest{Key: key, Value: newValue,
				OldVersion: old_uuid, NewVersion: new_uuid})

			switch {
			case errors.As(err, &authErr):
				return &kvapi.APIError{Method: "set", Err: authErr}
			case errors.As(err, &conflictErr):
				if conflictErr.ExpectedVersion == new_uuid {
					return nil
				}
				continue OUTER_FOR
			case errors.Is(err, kvapi.ErrKeyNotFound):
				updnewValue, err := updateFn(nil)
				if err != nil {
					return err
				}
				newValue = updnewValue
				old_uuid = uuid.UUID{}
				new_uuid = uuid.Must(uuid.NewV4())
				continue
			case err != nil:
				continue
			default:
				return nil
			}
		}

	}
}
