package main

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/shopspring/decimal"
)

var errInsufficientFunds = errors.New("insufficient funds")
var fee decimal.Decimal = decimal.NewFromFloat(1.001)

var marketPrices = map[string]decimal.Decimal{
	"btc": decimal.New(10000, 0),
	"eth": decimal.New(250, 0),
}

var order = [...]string{
	"btc",
	"eth",
}

func main() {
	cp := copyPortfolio{followers: make([]trader, 1, 4)}
	cp.init()

	cp.leaderDepositBase(decimal.New(10000, 0))
	cp.followerDepositBase(0, decimal.New(1000, 0))

	fmt.Println("Portfolio Start:")
	fmt.Println(&cp)

	per := decimal.New(100, 0)

	for cnt := 0; cnt < 100; cnt++ {
		if cnt == 30 {
			nf := cp.joinNewFollower()
			cp.followerDepositBase(nf, decimal.New(rand.Int63n(10000)+10, 0))
		} else if cnt == 60 {
			nf := cp.joinNewFollower()
			cp.followerDepositBase(nf, decimal.New(rand.Int63n(10000)+10, 0))
		}
		fmt.Println(&cp)
		if flipCoin() { // buy
			if flipCoin() { // btc
				canBuy := cp.leader.baseCurrency.Div(marketPrices["btc"]).Div(fee)
				if err := cp.leaderBuy("btc", decimal.NewFromFloat(float64(rand.Intn(100))).Mul(canBuy).Div(per)); err != nil {
					panic(err)
				}
			} else { // eth
				canBuy := cp.leader.baseCurrency.Div(marketPrices["eth"]).Div(fee)
				if err := cp.leaderBuy("eth", decimal.NewFromFloat(float64(rand.Intn(100))).Mul(canBuy).Div(per)); err != nil {
					panic(err)
				}
			}
		} else { // sell
			if flipCoin() { // btc
				canSell := cp.leader.openTrades["btc"].amount
				if canSell.IsZero() {
					continue
				}
				if err := cp.leaderSell("btc", decimal.NewFromFloat(float64(rand.Intn(100))).Mul(canSell).Div(per)); err != nil {
					panic(err)
				}
			} else { // eth
				canSell := cp.leader.openTrades["eth"].amount
				if canSell.IsZero() {
					continue
				}
				if err := cp.leaderSell("eth", decimal.NewFromFloat(float64(rand.Intn(100))).Mul(canSell).Div(per)); err != nil {
					panic(err)
				}
			}
		}

		marketPrices["btc"] = marketPrices["btc"].Mul(decimal.New(9995+rand.Int63n(10), 0).Div(decimal.New(10000, 0)))
		marketPrices["eth"] = marketPrices["eth"].Mul(decimal.New(9995+rand.Int63n(10), 0).Div(decimal.New(10000, 0)))
	}
	fmt.Println(&cp)

	/*if err := cp.leaderBuy("btc", decimal.NewFromFloat(0.5)); err != nil {
		panic(err)
	}

	fmt.Println("Leader bought 0.5BTC @", marketPrices["btc"].StringFixedBank(2), ":")
	fmt.Println(&cp)

	nf := cp.joinNewFollower()
	cp.followerDepositBase(nf, decimal.NewFromFloat(100))

	// change btc price
	marketPrices["btc"] = decimal.New(12000, 0)

	fmt.Println("BTC Price changes to", marketPrices["btc"].StringFixedBank(2), ":")
	fmt.Println(&cp)

	if err := cp.leaderSell("btc", decimal.NewFromFloat(0.3)); err != nil {
		panic(err)
	}

	fmt.Println("Leader sells 0.3BTC @", marketPrices["btc"].StringFixedBank(2), ":")
	fmt.Println(&cp)

	if err := cp.leaderBuy("eth", decimal.NewFromFloat(20)); err != nil {
		panic(err)
	}

	fmt.Println("Leader buys 20ETH @", marketPrices["eth"].StringFixedBank(2), ":")
	fmt.Println(&cp)

	marketPrices["btc"] = decimal.New(100000, 0)
	marketPrices["eth"] = decimal.New(1, 0)

	fmt.Println("BTC Price changes to", marketPrices["btc"].StringFixedBank(2), ":")
	fmt.Println("ETH Price changes to", marketPrices["eth"].StringFixedBank(2), ":")
	fmt.Println(&cp)

	marketPrices["btc"] = decimal.New(1, 0)
	marketPrices["eth"] = decimal.New(100000, 0)

	fmt.Println("BTC Price changes to", marketPrices["btc"].StringFixedBank(2), ":")
	fmt.Println("ETH Price changes to", marketPrices["eth"].StringFixedBank(2), ":")
	fmt.Println(&cp)

	fmt.Println("Looks like model holds, i.e. %P of leader in base always less or equal to that of any follower")*/
}

func (cp *copyPortfolio) leaderDepositBase(amount decimal.Decimal) {
	cp.leader.baseCurrency = cp.leader.baseCurrency.Add(amount)
}

func (cp *copyPortfolio) followerDepositBase(follower int, amount decimal.Decimal) {
	cp.followers[follower].baseCurrency = cp.followers[follower].baseCurrency.Add(amount)
}

func (cp *copyPortfolio) leaderBuy(asset string, amount decimal.Decimal) error {
	subTotal := amount.Mul(marketPrices[asset])
	totalCost := subTotal.Mul(fee)

	if cp.leader.baseCurrency.LessThan(totalCost) {
		return errInsufficientFunds
	}

	// Calc fraction of portfolio value going to be moved from base to asset
	frac := subTotal.Div(cp.leader.calcTotalPortValue())

	cp.leader.buy(asset, amount, totalCost)
	return cp.followersCopyBuy(asset, frac)
}

func (cp *copyPortfolio) leaderSell(asset string, amount decimal.Decimal) error {
	if amount.GreaterThan(cp.leader.openTrades[asset].amount) {
		return errInsufficientFunds
	}

	// Calc fraction of asset that is being closed of entire asset position
	frac := amount.Div(cp.leader.openTrades[asset].amount)

	cp.leader.sell(asset, amount)
	return cp.followersCopySell(asset, frac)
}

// Buys are done on fraction of total portfolio used (base to asset), not fraction of base used!
func (cp *copyPortfolio) followersCopyBuy(asset string, fracOfTotalPortfolioValueUsed decimal.Decimal) error {
	for i := range cp.followers {
		f := &cp.followers[i]

		subTotal := f.calcTotalPortValue().Mul(fracOfTotalPortfolioValueUsed)
		totalCost := subTotal.Mul(fee)

		if f.baseCurrency.LessThan(totalCost) {
			return errInsufficientFunds
		}

		amountToBuy := subTotal.Div(marketPrices[asset])
		f.buy(asset, amountToBuy, totalCost)
	}

	return nil
}

func (cp *copyPortfolio) followersCopySell(asset string, fracOfTotalAssetSold decimal.Decimal) error {
	for i := range cp.followers {
		f := &cp.followers[i]
		amountToSell := f.openTrades[asset].amount.Mul(fracOfTotalAssetSold)
		if amountToSell.Equal(decimal.Zero) {
			continue // if follower didn't copy an open trade, there is nothing to sell
		}
		f.sell(asset, amountToSell)
	}

	return nil
}

func flipCoin() bool {
	r := rand.Intn(10000)
	return r > 5000
}
