package api

import (
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	googlefirestore "cloud.google.com/go/firestore"
	"github.com/go-chi/chi"
	"github.com/navikt/datakatalogen/backend/firestore"
	log "github.com/sirupsen/logrus"
)

const (
	UserType           = "user"
	ServiceAccountType = "serviceAccount"
)

type AccessSubject struct {
	Subject string    `json:"subject" validate:"required"`
	Type    string    `json:"type" validate:"required"`
	Expires time.Time `json:"expires"`
}

func (a *api) getAccessUpdatesForProduct(w http.ResponseWriter, r *http.Request) {
	dpID := chi.URLParam(r, "productID")

	updates, err := a.firestore.GetAccessUpdatesForDataproduct(r.Context(), dpID)
	if err != nil {
		log.Errorf("Getting access updates for dataproduct: %v", err)
		respondf(w, http.StatusInternalServerError, "uh oh\n")
		return
	}

	if err := json.NewEncoder(w).Encode(updates); err != nil {
		log.Errorf("Serializing updateResponses: %v", err)
		respondf(w, http.StatusInternalServerError, "unable to serialize updateResponses\n")
		return
	}
}

func (a *api) removeProductAccess(w http.ResponseWriter, r *http.Request) {
	dpID := chi.URLParam(r, "productID")

	dp, err := a.firestore.GetDataproduct(r.Context(), dpID)
	if err != nil {
		log.Errorf("Getting dataproduct: %v", err)
		if status.Code(err) == codes.NotFound {
			respondf(w, http.StatusNotFound, "not found\n")
		} else {
			respondf(w, http.StatusBadRequest, "unable to get document\n")
		}
		return
	}

	var accessSubject AccessSubject
	if err := json.NewDecoder(r.Body).Decode(&accessSubject); err != nil {
		log.Errorf("Deserializing request document: %v", err)
		respondf(w, http.StatusBadRequest, "unable to deserialize request document\n")
		return
	}

	var subject string
	switch accessSubject.Type {
	case UserType:
		subject = "user:" + accessSubject.Subject
	case ServiceAccountType:
		subject = "serviceAccount:" + accessSubject.Subject
	default:
		{
			log.Errorf("Invalid AccessSubject.Type: %v", accessSubject.Type)
			respondf(w, http.StatusBadRequest, "invalid AccessSubject.Type\n")
			return
		}
	}

	requester := r.Context().Value("preferred_username").(string)
	requesterMember := r.Context().Value("member_name").(string)
	requesterGroups := r.Context().Value("teams").([]string)

	if contains(requesterGroups, dp.Dataproduct.Team) || accessSubject.Subject == requesterMember {
		_, ok := dp.Dataproduct.Access[subject]
		if !ok {
			log.Errorf("Requested subject does have an access entry")
			respondf(w, http.StatusBadRequest, "requested subject does not have an access entry")
			return
		}

		delete(dp.Dataproduct.Access, subject)
		dp.DocRef.Update(r.Context(), []googlefirestore.Update{{
			Path:  "access",
			Value: dp.Dataproduct.Access,
		}})

		log.Debugf("Revoking access for %v on datastore: %+v", subject, dp.Dataproduct.Datastore)

		if err := a.iam.RemoveDatastoreAccess(r.Context(), dp.Dataproduct.Datastore[0], subject); err != nil {
			log.Errorf("Removing datastore access: %v", err)
			respondf(w, http.StatusInternalServerError, "Could not revoke datastore access: %v\n", err)
			return
		}

		update := firestore.Delete(requester, dp.ID, accessSubject.Subject)
		if err := a.firestore.AddAccessUpdate(r.Context(), update); err != nil {
			log.Errorf("Adding access update: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	log.Errorf("Requester is not authorized to make changes to this rule: product id: %v, requester: %v, subject: %v", dp.ID, requester, accessSubject.Subject)
	respondf(w, http.StatusUnauthorized, "you are unauthorized to make changes to this access rule")
}

func (a *api) grantProductAccess(w http.ResponseWriter, r *http.Request) {
	dpID := chi.URLParam(r, "productID")

	dp, err := a.firestore.GetDataproduct(r.Context(), dpID)
	if err != nil {
		log.Errorf("Getting dataproduct: %v", err)
		if status.Code(err) == codes.NotFound {
			respondf(w, http.StatusNotFound, "not found\n")
		} else {
			respondf(w, http.StatusBadRequest, "unable to get document\n")
		}
		return
	}

	if len(dp.Dataproduct.Datastore) == 0 {
		log.Errorf("No datastore associated with dataproduct: %v (%v)", dp.ID, dp.Dataproduct.Name)
		respondf(w, http.StatusBadRequest, "no datastore associated with dataproduct: %v (%v)\n", dp.ID, dp.Dataproduct.Name)
		return
	}

	var accessSubject AccessSubject
	if err := json.NewDecoder(r.Body).Decode(&accessSubject); err != nil {
		log.Errorf("Deserializing request document: %v", err)
		respondf(w, http.StatusBadRequest, "unable to deserialize request document\n")
		return
	}

	if err := a.validate.Struct(accessSubject); err != nil {
		log.Errorf("Validating request document: %v", err)
		respondf(w, http.StatusBadRequest, "unable to validate request document\n")
		return
	}

	if accessSubject.Expires.Before(time.Now()) && !accessSubject.Expires.IsZero() {
		log.Errorf("Invalid AccessSubject.Expires: %v is already an expired time", accessSubject.Expires)
		respondf(w, http.StatusBadRequest, "invalid AccessSubject.Expires\n")
		return
	}

	var subject string
	switch accessSubject.Type {
	case UserType:
		subject = "user:" + accessSubject.Subject
	case ServiceAccountType:
		subject = "serviceAccount:" + accessSubject.Subject
	default:
		{
			log.Errorf("Invalid AccessSubject.Type: %v", accessSubject.Type)
			respondf(w, http.StatusBadRequest, "invalid AccessSubject.Type\n")
			return
		}
	}

	requester := r.Context().Value("preferred_username").(string)

	dp.Dataproduct.Access[subject] = accessSubject.Expires
	dp.DocRef.Update(r.Context(), []googlefirestore.Update{{
		Path:  "access",
		Value: dp.Dataproduct.Access,
	}})

	newAccess := map[string]time.Time{
		subject: accessSubject.Expires,
	}

	log.Debugf("Granting access for %v on datastore: %+v", subject, dp.Dataproduct.Datastore)

	if err := a.iam.UpdateDatastoreAccess(r.Context(), dp.Dataproduct.Datastore[0], newAccess); err != nil {
		log.Errorf("Granting datastore access: %v", err)
		respondf(w, http.StatusInternalServerError, "Could not grant datastore access: %v\n", err)
		return
	}

	update := firestore.Grant(requester, dp.ID, accessSubject.Subject, accessSubject.Expires)
	if err := a.firestore.AddAccessUpdate(r.Context(), update); err != nil {
		log.Errorf("Adding access update: %v", err)
	}
}
