package accessensurer

import (
	"context"
	"fmt"
	"time"

	googlefirestore "cloud.google.com/go/firestore"
	"github.com/navikt/datakatalogen/backend/firestore"

	"github.com/navikt/datakatalogen/backend/config"
	"github.com/navikt/datakatalogen/backend/iam"
	log "github.com/sirupsen/logrus"
)

const AccessEnsurance2000 = "AccessEnsurance2000"

type AccessEnsurer struct {
	ctx             context.Context
	iam             *iam.Client
	firestore       *firestore.Firestore
	cfg             config.Config
	updateFrequency time.Duration
}

func New(ctx context.Context, cfg config.Config, firestore *firestore.Firestore, iamClient *iam.Client, updateFrequency time.Duration) *AccessEnsurer {
	return &AccessEnsurer{
		ctx:             ctx,
		firestore:       firestore,
		iam:             iamClient,
		cfg:             cfg,
		updateFrequency: updateFrequency,
	}
}
func (a *AccessEnsurer) Run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Debugf("Checking access...")
			if err := a.ensureAccesses(); err != nil {
				log.Errorf("Checking access: %v", err)
			}
			ticker.Reset(a.updateFrequency)
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *AccessEnsurer) ensureAccesses() error {
	dataproducts, err := a.firestore.GetDataproducts(a.ctx)
	if err != nil {
		return fmt.Errorf("getting dataproducts: %w", err)
	}

	for _, dataproduct := range dataproducts {
		if err := a.checkAccess(dataproduct); err != nil {
			return err
		}
	}

	return nil
}

func (a *AccessEnsurer) checkAccess(dataproduct *firestore.DataproductResponse) error {
	if len(dataproduct.Dataproduct.Datastore) == 0 {
		// we have no access to check here
		return nil
	}
	datastore := dataproduct.Dataproduct.Datastore[0]
	toDelete := make([]string, 0)

	for subject, expiry := range dataproduct.Dataproduct.Access {
		log.Debugf("Ensuring access for %v with expiry %v", subject, expiry)
		if expiry.IsZero() {
			log.Infof("Ensuring access for %v in %v, zero-value expiry means it should last forever", subject, datastore["type"])
			err := a.ensureAccess(dataproduct.ID, datastore, subject, expiry)
			if err != nil {
				return err
			}
			continue
		}
		if expiry.Before(time.Now()) {
			log.Infof("Access expired, removing %v from %v", subject, datastore["type"])
			if err := a.iam.RemoveDatastoreAccess(a.ctx, datastore, subject); err != nil {
				return err
			}

			deletion := firestore.Delete(AccessEnsurance2000, dataproduct.ID, subject)

			if err := a.firestore.AddAccessUpdate(a.ctx, deletion); err != nil {
				log.Errorf("Adding access update: %v", err)
			}

			toDelete = append(toDelete, subject)
		} else {
			err := a.ensureAccess(dataproduct.ID, datastore, subject, expiry)
			if err != nil {
				return err
			}
		}
	}

	if len(toDelete) > 0 {
		for _, subject := range toDelete {
			delete(dataproduct.Dataproduct.Access, subject)
		}
		if _, err := dataproduct.DocRef.Update(a.ctx, []googlefirestore.Update{{
			Path:  "access",
			Value: dataproduct.Dataproduct.Access,
		}}); err != nil {
			log.Errorf("Updating access for dataproduct: %v", err)
		}
	}

	update := firestore.Verify(AccessEnsurance2000, dataproduct.ID)
	if err := a.firestore.AddAccessUpdate(a.ctx, update); err != nil {
		log.Errorf("Adding access update: %v", err)
	}
	return nil
}

func (a *AccessEnsurer) ensureAccess(dataproductID string, datastore map[string]string, subject string, expiry time.Time) error {
	access, err := a.iam.CheckDatastoreAccess(a.ctx, datastore, subject)
	if err != nil {
		return err
	}
	if !access {
		log.Infof("Access state out of sync with Google %v, giving access to %v", datastore["type"], subject)
		accessMap := map[string]time.Time{subject: expiry}
		if err := a.iam.UpdateDatastoreAccess(a.ctx, datastore, accessMap); err != nil {
			return err
		}

		update := firestore.Grant(AccessEnsurance2000, dataproductID, subject, expiry)
		if err := a.firestore.AddAccessUpdate(a.ctx, update); err != nil {
			log.Errorf("Adding access update: %v", err)
		}
	}
	return nil
}
