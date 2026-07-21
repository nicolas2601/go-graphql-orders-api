package graphql

import (
	"github.com/nicolas2601/go-graphql-orders-api/internal/delivery/graphql/model"
	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
	"github.com/nicolas2601/go-graphql-orders-api/internal/usecase"
)

// Los tiempos se normalizan a UTC para que el scalar Time los serialice de forma consistente.

func toUserModel(u domain.User) *model.User {
	return &model.User{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt.UTC()}
}

func toProductModel(p domain.Product) *model.Product {
	return &model.Product{ID: p.ID, Name: p.Name, Price: p.Price, Stock: p.Stock, CreatedAt: p.CreatedAt.UTC()}
}

func toOrderModel(o domain.Order) *model.Order {
	items := make([]*model.OrderItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, &model.OrderItem{ProductID: it.ProductID, Quantity: it.Quantity, UnitPrice: it.UnitPrice})
	}
	return &model.Order{
		ID:        o.ID,
		UserID:    o.UserID,
		Items:     items,
		Total:     o.Total,
		Status:    model.OrderStatus(o.Status),
		CreatedAt: o.CreatedAt.UTC(),
	}
}

func toAuthPayload(pair usecase.TokenPair, user domain.User) *model.AuthPayload {
	return &model.AuthPayload{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, User: toUserModel(user)}
}

func toProductPage(page usecase.Page[domain.Product]) *model.ProductPage {
	items := make([]*model.Product, 0, len(page.Items))
	for _, p := range page.Items {
		items = append(items, toProductModel(p))
	}
	return &model.ProductPage{
		Items:       items,
		Total:       page.Total,
		Page:        page.Page,
		PageSize:    page.PageSize,
		HasNextPage: page.HasNextPage(),
	}
}

func toOrderPage(page usecase.Page[domain.Order]) *model.OrderPage {
	items := make([]*model.Order, 0, len(page.Items))
	for _, o := range page.Items {
		items = append(items, toOrderModel(o))
	}
	return &model.OrderPage{
		Items:       items,
		Total:       page.Total,
		Page:        page.Page,
		PageSize:    page.PageSize,
		HasNextPage: page.HasNextPage(),
	}
}

func toOrderLines(inputs []*model.OrderItemInput) []usecase.OrderLine {
	lines := make([]usecase.OrderLine, 0, len(inputs))
	for _, in := range inputs {
		lines = append(lines, usecase.OrderLine{ProductID: in.ProductID, Quantity: in.Quantity})
	}
	return lines
}

func toDomainFilter(f *model.ProductFilter) domain.ProductFilter {
	if f == nil {
		return domain.ProductFilter{}
	}
	return domain.ProductFilter{Name: f.Name, MinPrice: f.MinPrice, MaxPrice: f.MaxPrice}
}

func pageOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
