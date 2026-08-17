package settings

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetShopSettings() (*ShopSettings, error) {
	name, err := s.repo.Get(KeyShopName)
	if err != nil {
		return nil, err
	}
	phone, err := s.repo.Get(KeyShopPhone)
	if err != nil {
		return nil, err
	}
	currency, err := s.repo.Get(KeyCurrency)
	if err != nil {
		return nil, err
	}
	return &ShopSettings{Name: name, Phone: phone, Currency: currency}, nil
}

func (s *Service) GetShopSettingsDisplay() (*ShopSettings, error) {
	shop, err := s.GetShopSettings()
	if err != nil {
		return nil, err
	}
	if shop.Currency == "" {
		shop.Currency = "VND"
	}
	return shop, nil
}

func (s *Service) UpdateShopSettings(req UpdateSettingsRequest) (*ShopSettings, error) {
	if req.Name != nil {
		if err := s.repo.Set(KeyShopName, *req.Name); err != nil {
			return nil, err
		}
	}
	if req.Phone != nil {
		if err := s.repo.Set(KeyShopPhone, *req.Phone); err != nil {
			return nil, err
		}
	}
	if req.Currency != nil {
		if err := s.repo.Set(KeyCurrency, *req.Currency); err != nil {
			return nil, err
		}
	}
	return s.GetShopSettingsDisplay()
}

func (s *Service) SeedDefaults(name, phone, currency string) error {
	if name != "" {
		v, err := s.repo.Get(KeyShopName)
		if err != nil {
			return err
		}
		if v == "" {
			if err := s.repo.Set(KeyShopName, name); err != nil {
				return err
			}
		}
	}
	if phone != "" {
		v, err := s.repo.Get(KeyShopPhone)
		if err != nil {
			return err
		}
		if v == "" {
			if err := s.repo.Set(KeyShopPhone, phone); err != nil {
				return err
			}
		}
	}
	v, err := s.repo.Get(KeyCurrency)
	if err != nil {
		return err
	}
	if v == "" {
		if currency == "" {
			currency = "VND"
		}
		if err := s.repo.Set(KeyCurrency, currency); err != nil {
			return err
		}
	}
	return nil
}
