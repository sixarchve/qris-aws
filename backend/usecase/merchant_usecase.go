package usecase

import (
	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/domain/repository"
)

type MerchantUsecase interface {
	GetActiveMerchants() ([]entity.Merchant, error)
}

type merchantUsecase struct {
	repo repository.MerchantRepository
}

func NewMerchantUsecase(repo repository.MerchantRepository) MerchantUsecase {
	return &merchantUsecase{repo: repo}
}

func (u *merchantUsecase) GetActiveMerchants() ([]entity.Merchant, error) {
	return u.repo.FindAllActive()
}
