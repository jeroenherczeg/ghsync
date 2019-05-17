package ghsync

import (
	"context"
	"database/sql"

	"github.com/src-d/ghsync/models"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-kallax.v1"
	"gopkg.in/src-d/go-queue.v1"
)

type PullRequestSyncer struct {
	s *models.PullRequestStore
	c *github.Client
}

func NewPullRequestSyncer(db *sql.DB, c *github.Client) *PullRequestSyncer {
	return &PullRequestSyncer{
		s: models.NewPullRequestStore(db),
		c: c,
	}
}

func (s *PullRequestSyncer) QueueRepository(q queue.Queue, owner, repo string) error {
	opts := &github.PullRequestListOptions{}
	opts.ListOptions.PerPage = 10
	opts.State = "all"

	for {
		requests, r, err := s.c.PullRequests.List(context.TODO(), owner, repo, opts)
		if err != nil {
			return err
		}

		for _, r := range requests {
			j, err := NewPullRequestSyncJob(owner, repo, r.GetNumber())
			if err != nil {
				return err
			}

			if err := q.Publish(j); err != nil {
				return err
			}
		}

		if r.NextPage == 0 {
			break
		}

		opts.Page = r.NextPage
	}

	return nil
}

func (s *PullRequestSyncer) Sync(owner string, repo string, number int) error {
	pr, _, err := s.c.PullRequests.Get(context.TODO(), owner, repo, number)
	if err != nil {
		return err
	}

	record, err := s.s.FindOne(models.NewPullRequestQuery().
		Where(kallax.And(
			kallax.Eq(models.Schema.PullRequest.ID, pr.GetID()),
		)),
	)
	if record == nil {
		record = models.NewPullRequest()
		record.PullRequest = *pr

		return s.s.Insert(record)
	}

	record.PullRequest = *pr
	_, err = s.s.Update(record)
	return err

}
