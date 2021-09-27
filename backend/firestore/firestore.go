package firestore

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

const (
	DeleteType = "delete"
	GrantType  = "grant"
	VerifyType = "verify"
)

type Firestore struct {
	dataproducts  *firestore.CollectionRef
	accessUpdates *firestore.CollectionRef
}

type Dataproduct struct {
	Name        string               `firestore:"name" json:"name,omitempty"`
	Description string               `firestore:"description" json:"description,omitempty"`
	Datastore   []map[string]string  `firestore:"datastore" json:"datastore,omitempty"`
	Team        string               `firestore:"team" json:"team,omitempty"`
	Access      map[string]time.Time `firestore:"access" json:"access,omitempty"`
}

type DataproductResponse struct {
	ID          string                 `json:"id"`
	Dataproduct Dataproduct            `json:"data_product"`
	Updated     time.Time              `json:"updated"`
	Created     time.Time              `json:"created"`
	DocRef      *firestore.DocumentRef `json:"-"`
}

type AccessUpdate struct {
	ProductID  string    `firestore:"dataproduct_id" json:"dataproduct_id"`
	Author     string    `firestore:"author" json:"author"`
	Subject    string    `firestore:"subject" json:"subject"`
	Action     string    `firestore:"action" json:"action"`
	UpdateTime time.Time `firestore:"time" json:"time"`
	Expires    time.Time `firestore:"expires" json:"expires"`
}

func New(ctx context.Context, googleProjectID, dataproductCollection, accessUpdatesCollection string) (*Firestore, error) {
	client, err := firestore.NewClient(ctx, googleProjectID)

	if err != nil {
		return nil, fmt.Errorf("initializing firestore client: %v", err)
	}
	return &Firestore{
		dataproducts:  client.Collection(dataproductCollection),
		accessUpdates: client.Collection(accessUpdatesCollection),
	}, nil
}

func (f *Firestore) CreateDataproduct(ctx context.Context, dp Dataproduct) (string, error) {
	ref, _, err := f.dataproducts.Add(ctx, dp)
	if err != nil {
		log.Errorf("Adding dataproduct to collection: %v", err)
		return "", fmt.Errorf("adding dataproduct to collection: %w", err)
	}
	return ref.ID, nil
}

func (f *Firestore) GetDataproduct(ctx context.Context, id string) (*DataproductResponse, error) {
	doc, err := f.dataproducts.Doc(id).Get(ctx)
	if err != nil {
		log.Errorf("Getting dataproduct from collection: %v", err)
		return nil, fmt.Errorf("getting dataproduct from collection: %w", err)
	}

	return toResponse(doc)
}

func (f *Firestore) GetDataproducts(ctx context.Context) ([]*DataproductResponse, error) {
	dataproducts := []*DataproductResponse{}

	iter := f.dataproducts.Documents(ctx)
	defer iter.Stop()
	for {
		document, err := iter.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			log.Errorf("Iterating documents: %v", err)
			break
		}

		dpr, err := toResponse(document)

		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			log.Errorf("Creating DataproductResponse: %v", err)
			return nil, fmt.Errorf("creating DataproductResponse: %w", err)
		}

		dataproducts = append(dataproducts, dpr)
	}

	return dataproducts, nil
}

func (f *Firestore) UpdateDataproduct(ctx context.Context, id string, new Dataproduct) error {
	old, err := f.GetDataproduct(ctx, id)
	if err != nil {
		return fmt.Errorf("getting dataproduct: %w", err)
	}

	_, err = old.DocRef.Update(ctx, createUpdates(new, old.Dataproduct.Team, old.Dataproduct.Access))
	if err != nil {
		return fmt.Errorf("updating dataproduct document: %w", err)
	}

	log.Debugf("Updated dataproduct: %v", id)

	return nil
}

func (f *Firestore) DeleteDataproduct(ctx context.Context, id string) error {
	documentRef := f.dataproducts.Doc(id)

	if _, err := documentRef.Delete(ctx); err != nil {
		return fmt.Errorf("deleting firestore document: %w", err)
	}

	return nil
}

func (f *Firestore) AddAccessUpdate(ctx context.Context, accessUpdate AccessUpdate) error {
	if _, _, err := f.accessUpdates.Add(ctx, accessUpdate); err != nil {
		return fmt.Errorf("adding access update: %w", err)
	}
	return nil
}

func (f *Firestore) GetAccessUpdatesForDataproduct(ctx context.Context, id string) ([]AccessUpdate, error) {
	var accessUpdates []AccessUpdate

	query := f.accessUpdates.Where("dataproduct_id", "==", id).OrderBy("time", firestore.Desc)
	iter := query.Documents(ctx)
	defer iter.Stop()

	for {
		document, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Errorf("Iterating documents: %v", err)
			break
		}

		var update AccessUpdate
		if err := document.DataTo(&update); err != nil {
			log.Errorf("Deserializing firestore document: %v", err)
			return nil, fmt.Errorf("deserializing firestore document: %w", err)
		}

		accessUpdates = append(accessUpdates, update)
	}

	return accessUpdates, nil
}

func createUpdates(dp Dataproduct, currentTeam string, access map[string]time.Time) (updates []firestore.Update) {
	if len(dp.Name) > 0 {
		updates = append(updates, firestore.Update{
			Path:  "name",
			Value: dp.Name,
		})
	}
	if len(dp.Description) > 0 {
		updates = append(updates, firestore.Update{
			Path:  "description",
			Value: dp.Description,
		})
	}
	if len(dp.Datastore) > 0 {
		updates = append(updates, firestore.Update{
			Path:  "datastore",
			Value: dp.Datastore,
		})
	}
	if len(dp.Team) > 0 {
		updates = append(updates, firestore.Update{
			Path:  "team",
			Value: dp.Team,
		})

		delete(access, fmt.Sprintf("group:%v@nav.no", currentTeam))
		access[fmt.Sprintf("group:%v@nav.no", dp.Team)] = time.Time{}
		updates = append(updates, firestore.Update{
			Path:  "access",
			Value: access,
		})
	}

	return
}

func toResponse(document *firestore.DocumentSnapshot) (*DataproductResponse, error) {
	var dp Dataproduct

	if err := document.DataTo(&dp); err != nil {
		return nil, fmt.Errorf("populating fields in dataproduct struct: %w", err)
	}

	if len(dp.Name) == 0 { // empty/invalid dataproduct. This is known to happen during creation of Firestore collections
		return nil, status.Errorf(codes.NotFound, "no valid dataproduct in path: %v", document.Ref.Path)
	}

	var dpr DataproductResponse
	dpr.Dataproduct = dp
	dpr.ID = document.Ref.ID
	dpr.Updated = document.UpdateTime
	dpr.Created = document.CreateTime
	dpr.DocRef = document.Ref

	return &dpr, nil
}

func Delete(author, productID, subject string) (au AccessUpdate) {
	au.Action = DeleteType
	au.UpdateTime = time.Now()
	au.ProductID = productID
	au.Subject = subject
	au.Author = author
	return
}

func Grant(author, productID, subject string, expiry time.Time) (au AccessUpdate) {
	au.Action = GrantType
	au.UpdateTime = time.Now()
	au.Expires = expiry
	au.ProductID = productID
	au.Subject = subject
	au.Author = author
	return
}

func Verify(author, productID string) (au AccessUpdate) {
	au.Action = VerifyType
	au.UpdateTime = time.Now()
	au.ProductID = productID
	au.Author = author
	return
}
