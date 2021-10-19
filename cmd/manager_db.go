package main

import (
	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	"log"
)

// runnerDB implements runner.DataSource over the primary
// database.
type runnerDB struct {
	queries *Queries
	logger  *log.Logger
}

func newManagerDB(q *Queries, l *log.Logger) *runnerDB {
	return &runnerDB{
		queries: q,
		logger:  l,
	}
}

// NextCampaigns retrieves active campaigns ready to be processed.
func (r *runnerDB) NextCampaigns(excludeIDs []int64) ([]*models.Campaign, error) {
	var out []*models.Campaign
	err := r.queries.NextCampaigns.Select(&out, pq.Int64Array(excludeIDs))
	return out, err
}

// NextSubscribers retrieves a subset of subscribers of a given campaign.
// Since batches are processed sequentially, the retrieval is ordered by ID,
// and every batch takes the last ID of the last batch and fetches the next
// batch above that.
func (r *runnerDB) NextSubscribers(campID, limit int) ([]models.Subscriber, error) {
	var out []models.Subscriber
	//defer func(begin time.Time) {
	//	r.logger.Printf(" NextSubscribers campID: %s , took: %v, totalCamp: %v", campID, time.Since(begin), len(out))
	//}(time.Now())
	err := r.queries.NextCampaignSubscribers.Select(&out, campID)
	return out, err
}

// GetCampaign fetches a campaign from the database.
func (r *runnerDB) GetCampaign(campID int) (*models.Campaign, error) {
	var out = &models.Campaign{}
	err := r.queries.GetCampaign.Get(out, campID, nil)
	return out, err
}

// UpdateCampaignStatus updates a campaign's status.
func (r *runnerDB) UpdateCampaignStatus(campID int, status string) error {
	_, err := r.queries.UpdateCampaignStatus.Exec(campID, status)
	return err
}

// CreateLink registers a URL with a UUID for tracking clicks and returns the UUID.
func (r *runnerDB) CreateLink(url string) (string, error) {
	// Create a new UUID for the URL. If the URL already exists in the DB
	// the UUID in the database is returned.
	uu, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	var out string
	if err := r.queries.CreateLink.Get(&out, uu, url); err != nil {
		return "", err
	}

	return out, nil
}

// UpdateCampaignStatus updates a campaign's status.
func (r *runnerDB) UpdateLastEmailSent(email string) error {
	_, err := r.queries.UpdateLastEmailSent.Exec(email)
	return err
}

func (r *runnerDB) UpdateSentCampaign(campID, limit, lastSubsId int) error {
	//defer func(begin time.Time) {
	//	r.logger.Printf(" UpdateSentCampaign campID: %s, limit: %s, lastSubsId: %s, took: %v", campID, limit, lastSubsId, time.Since(begin))
	//}(time.Now())
	_, err := r.queries.UpdateSendCampaignCounts.Exec(campID, limit, lastSubsId)
	if err != nil {
		r.logger.Printf(" Error UpdateSendCampaignCounts: ", err)
		return err
	}
	_, err = r.queries.UpdateSettingCampaignCounts.Exec(limit)
	if err != nil {
		r.logger.Printf(" Error UpdateSettingCampaignCounts: ", err)
		return err
	}
	return err
}
