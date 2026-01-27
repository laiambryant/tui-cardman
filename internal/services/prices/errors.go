package prices

import "fmt"

type FailedToDeleteTCGPlayerPricesError struct {
	Err error
}

func (e *FailedToDeleteTCGPlayerPricesError) Error() string {
	return fmt.Sprintf("failed to delete TCGPlayer prices: %v", e.Err)
}

func (e *FailedToDeleteTCGPlayerPricesError) Unwrap() error {
	return e.Err
}

type FailedToInsertTCGPlayerPriceError struct {
	Err error
}

func (e *FailedToInsertTCGPlayerPriceError) Error() string {
	return fmt.Sprintf("failed to insert TCGPlayer price: %v", e.Err)
}

func (e *FailedToInsertTCGPlayerPriceError) Unwrap() error {
	return e.Err
}

type FailedToDeleteCardMarketPricesError struct {
	Err error
}

func (e *FailedToDeleteCardMarketPricesError) Error() string {
	return fmt.Sprintf("failed to delete CardMarket prices: %v", e.Err)
}

func (e *FailedToDeleteCardMarketPricesError) Unwrap() error {
	return e.Err
}

type FailedToInsertCardMarketPriceError struct {
	Err error
}

func (e *FailedToInsertCardMarketPriceError) Error() string {
	return fmt.Sprintf("failed to insert CardMarket price: %v", e.Err)
}

func (e *FailedToInsertCardMarketPriceError) Unwrap() error {
	return e.Err
}
