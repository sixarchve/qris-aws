package postgres

import (
	"qris-latency-optimizer/domain/entity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	var err error
	DB, err = gorm.Open(postgres.Open(LoadDatabaseConfig()), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	if err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		panic(err)
	}

	var c entity.Merchant
	var d entity.Transaction

	if err := DB.AutoMigrate(&c, &d); err != nil {
		panic(err)
	}

	seedMerchants()
}

func seedMerchants() {
	merchants := []entity.Merchant{
		{QRID: "TEST001", MerchantName: "Kantin FILKOM UB", IsActive: true},
		{QRID: "TEST002", MerchantName: "TESTING STORE", IsActive: true},
	}

	for _, merchant := range merchants {
		if err := DB.Where("qr_id = ?", merchant.QRID).FirstOrCreate(&merchant).Error; err != nil {
			panic(err)
		}
	}
}
