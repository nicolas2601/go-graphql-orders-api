# Tasks

## 1. Errores tipados
- [ ] 1.1 Definir los sentinels de dominio en internal/domain/errors.go

## 2. Entidades e invariantes
- [ ] 2.1 Product: entidad + NewProduct + Validate (TDD)
- [ ] 2.2 User: entidad + NewUser + ValidateEmail + ValidatePassword (TDD)
- [ ] 2.3 Order + OrderItem: NewOrder (total, invariantes) + Confirm/Cancel (TDD)

## 3. Ports (interfaces del dominio)
- [ ] 3.1 UserRepository, ProductRepository, OrderRepository
- [ ] 3.2 TxManager, TokenService, PasswordHasher
