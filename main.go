package main

import (
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"math/rand"
	"strconv"
	"time"
)

var errInsufficientFunds = errors.New("insufficient funds")
var fee = decimal.New(1001, -3)

var marketPrices = map[string]decimal.Decimal{
	"btc": decimal.New(10000, 0),
	"eth": decimal.New(250, 0),
}

var order = [...]string{
	"btc",
	"eth",
}

func main() {
	// Comment out if you want deterministic/reproducible results.
	// Won't work in sandbox where time.Now() does not change, like go playground
	rand.Seed(time.Now().UnixNano())

	cp := copyPortfolio{followers: make([]trader, 1, 4)}
	cp.init()

	cp.leaderDepositBase(decimal.New(10000, 0))
	cp.followerDepositBase(0, decimal.New(1000, 0))

	fmt.Println("Portfolio Start:")
	fmt.Println(&cp)

	per := decimal.New(100, 0)

	iterations := 500
	for cnt := 0; cnt < iterations; cnt++ {
		fmt.Println("Iteration", cnt)
		if flipCoin() { // buy
			if flipCoin() { // btc
				canBuy := cp.leader.baseCurrency.Div(marketPrices["btc"]).Div(fee)
				toBuy := decimal.New(rand.Int63n(100), 0).Mul(canBuy).Div(per).Truncate(8)
				fmt.Println("Leader buying", toBuy.StringFixedBank(8), "BTC. Market Prices: BTC:", marketPrices["btc"].StringFixedBank(2), "ETH:", marketPrices["eth"].StringFixedBank(2), ":")
				if err := cp.leaderBuy("btc", toBuy); err != nil {
					panic(err)
				}
			} /*else { // eth
				canBuy := cp.leader.baseCurrency.Div(marketPrices["eth"]).Div(fee)
				toBuy := decimal.New(rand.Int63n(100), 0).Mul(canBuy).Div(per).Truncate(8)
				fmt.Println("Leader buying", toBuy.StringFixedBank(8), "ETH. Market Prices: BTC:", marketPrices["btc"].StringFixedBank(2), "ETH:", marketPrices["eth"].StringFixedBank(2), ":")
				if err := cp.leaderBuy("eth", toBuy); err != nil {
					panic(err)
				}
			}*/
		} else { // sell
			if flipCoin() { // btc
				if oT, ok := cp.leader.positions["btc"]; ok {
					canSell := oT.totalAmount
					if canSell.IsZero() {
						continue
					}

					toSell := decimal.New(rand.Int63n(100), 0).Mul(canSell).Div(per).Truncate(8)
					fmt.Println("Leader selling", toSell.StringFixedBank(8), "BTC. Market Prices: BTC:", marketPrices["btc"].StringFixedBank(2), "ETH:", marketPrices["eth"].StringFixedBank(2), ":")
					if err := cp.leaderSell("btc", toSell); err != nil {
						panic(err)
					}
				}

			} /*else { // eth
				if oT, ok := cp.leader.positions["eth"]; ok {
					canSell := oT.totalAmount
					if canSell.IsZero() {
						continue
					}

					toSell := decimal.New(rand.Int63n(100), 0).Mul(canSell).Div(per).Truncate(8)
					fmt.Println("Leader selling", toSell.StringFixedBank(8), "ETH. Market Prices: BTC:", marketPrices["btc"].StringFixedBank(2), "ETH:", marketPrices["eth"].StringFixedBank(2), ":")
					if err := cp.leaderSell("eth", toSell); err != nil {
						panic(err)
					}
				}
			}*/
		}

		marketPrices["btc"] = marketPrices["btc"].Mul(decimal.New(9995+rand.Int63n(10), 0).Div(decimal.New(10000, 0))).Truncate(4)
		marketPrices["eth"] = marketPrices["eth"].Mul(decimal.New(9995+rand.Int63n(10), 0).Div(decimal.New(10000, 0))).Truncate(4)

		if cnt == iterations/4 {
			nf := cp.joinNewFollower()
			cp.followerDepositBase(nf, decimal.New(rand.Int63n(10000)+10, 0))
		} else if cnt == iterations/2 {
			nf := cp.joinNewFollower()
			cp.followerDepositBase(nf, decimal.New(rand.Int63n(1000)+10, 0))
		} else if cnt == iterations*3/4 {
			nf := cp.joinNewFollower()
			cp.followerDepositBase(nf, decimal.New(rand.Int63n(100)+10, 0))
		}
		fmt.Println(&cp)

		// check if leader % of portfolio in base always less or equal to that of any follower
		leaderFrac := cp.leader.baseCurrency.Div(cp.leader.calcTotalPortValue())
		for i := range cp.followers {
			f := &cp.followers[i]

			followerFrac := f.baseCurrency.Div(f.calcTotalPortValue())
			if followerFrac.LessThan(leaderFrac) {
				fmt.Println("follower base frac:", followerFrac.StringFixedBank(17), "leader base frac:", leaderFrac.StringFixedBank(17), "diff:", leaderFrac.Sub(followerFrac).StringFixedBank(17))
				panic("Follower " + strconv.Itoa(i) + " broken")
			}
		}
	}

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
	if amount.GreaterThan(cp.leader.positions[asset].totalAmount) {
		return errInsufficientFunds
	}

	// Calc fraction of asset that is being closed of entire asset position
	frac := amount.Div(cp.leader.positions[asset].totalAmount)

	cp.leader.sell(asset, amount)
	profitFromSharing := cp.followersCopySell(asset, frac)
	if profitFromSharing.GreaterThan(decimal.Zero) {
		fmt.Println("Leader gets ", profitFromSharing.StringFixedBank(8), " shared profit for this sell")
	}
	return nil
}

// Buys are done on fraction of total portfolio used (base to asset), not fraction of base used!
func (cp *copyPortfolio) followersCopyBuy(asset string, fracOfTotalPortfolioValueUsed decimal.Decimal) error {
	for i := range cp.followers {
		f := &cp.followers[i]

		toBuyWith := f.calcTotalPortValue().Mul(fracOfTotalPortfolioValueUsed)
		amountToBuy := toBuyWith.Div(marketPrices[asset]).Truncate(8)

		subTotal := amountToBuy.Mul(marketPrices[asset])
		totalCost := subTotal.Mul(fee)

		if f.baseCurrency.LessThan(totalCost) {
			return errInsufficientFunds
		}

		f.buy(asset, amountToBuy, totalCost)
	}

	return nil
}

// Returns total profit leader made from all followers for this sell.
func (cp *copyPortfolio) followersCopySell(asset string, fracOfTotalAssetSold decimal.Decimal) decimal.Decimal {
	var leaderProfit decimal.Decimal

	for i := range cp.followers {
		f := &cp.followers[i]

		if oT, ok := f.positions[asset]; ok && !oT.totalAmount.IsZero() {
			amountToSell := oT.totalAmount.Mul(fracOfTotalAssetSold)
			fGains := f.sell(asset, amountToSell)
			if fGains.GreaterThan(decimal.Zero) { // share % of profit with leader if > 0

				// Assume follower profit split is 80F/20L
				leaderShare := fGains.Mul(decimal.New(2, -1))

				// calc what to take from follower base and asset
				ratio := oT.totalAmount.Mul(marketPrices[asset]).Div(f.calcTotalPortValue())

				oT.totalAmount = oT.totalAmount.Sub(ratio.Mul(leaderShare).Div(marketPrices[asset]))
				f.baseCurrency = f.baseCurrency.Sub((decimal.New(1, 0).Sub(ratio)).Mul(leaderShare))

				leaderProfit = leaderProfit.Add(leaderShare)
			}
		}
	}

	return leaderProfit
}

func flipCoin() bool {
	r := rand.Intn(10000)
	return r > 5000
}
