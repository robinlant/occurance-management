package service

import (
	"context"

	"github.com/robinlant/occurance-management/internal/domain"
	"github.com/robinlant/occurance-management/internal/repository"
)

type GroupService struct {
	groups repository.GroupRepository
}

func NewGroupService(groups repository.GroupRepository) *GroupService {
	return &GroupService{groups: groups}
}

func (s *GroupService) Create(ctx context.Context, name string) (domain.Group, error) {
	return s.groups.Save(ctx, domain.Group{Name: name})
}

func (s *GroupService) List(ctx context.Context) ([]domain.Group, error) {
	return s.groups.FindAll(ctx)
}

func (s *GroupService) GetByID(ctx context.Context, id int64) (domain.Group, error) {
	return s.groups.FindByID(ctx, id)
}

func (s *GroupService) Delete(ctx context.Context, id int64) error {
	return s.groups.Delete(ctx, id)
}
