package main

import (
	"github.com/go-pg/pg/v10"
	"log"
)

type ProductRepository struct {
	pgdb *pg.DB
}

func NewProductRepository(pgdb *pg.DB) *ProductRepository {
	pr := ProductRepository{
		pgdb: pgdb,
	}

	return &pr
}

func (pr *ProductRepository) Order(p *Product) (int, error) {
	res, err := pr.pgdb.Model(p).
		Set("quantity = product.Quantity - ?", p.Quantity).
		Where("id = ? AND quantity >= ?", p.Id, p.Quantity).
		Update()

	if nil != err {
		log.Printf("%+v", err.Error())
		return 0, err
	}

	return res.RowsAffected(), err
}

func (pr *ProductRepository) Store(p *Product) (int, error) {
	res, err := pr.pgdb.Model(p).
		OnConflict("(id) DO UPDATE").
		Set("quantity = product.Quantity + ?", p.Quantity).
		Returning("quantity").
		Insert()

	if nil != err {
		log.Printf("%+v", err.Error())
		return 0, err
	}

	return res.RowsAffected(), err
}

func (pr *ProductRepository) FindById(pId string) (*Product, error) {
	p := new(Product)
	err := pr.pgdb.Model(p).
		Where("id = ?", pId).
		Select()

	if nil != err {
		log.Printf("%+v", err.Error())
		return nil, err
	}

	return p, err
}
