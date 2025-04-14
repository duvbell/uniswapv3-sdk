package main

import (
	"context"
	"math/big"
	"os"
	"time"

	"log"

	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/daoleno/uniswapv3-sdk/examples/helper"
	"github.com/daoleno/uniswapv3-sdk/periphery"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
	client, err := ethclient.Dial(helper.PolygonRPC)
	if err != nil {
		log.Fatal(err)
	}
	wallet := helper.InitWallet(os.Getenv("PRIVATE_KEY_MAIN"))
	if wallet == nil {
		log.Fatal("init wallet failed")
	}

	// log out the wallet adderss
	log.Printf("wallet address: %s", wallet.PublicKey.String())

	// get the token address
	wip := coreEntities.NewToken(1, common.HexToAddress("0x1514000000000000000000000000000000000000"), 18, "weth", "wrapped ip")
	usdc := coreEntities.NewToken(1, common.HexToAddress("0xF1815bd50389c46847f0Bda824eC8da914045D14"), 6, "usdc", "usdc")
	pool, err := helper.ConstructV3Pool(client, wip, usdc, uint64(constants.FeeMedium))
	if err != nil {
		log.Fatal(err)
	}

	//0.01%
	slippageTolerance := coreEntities.NewPercent(big.NewInt(1), big.NewInt(1000))
	//after 5 minutes
	d := time.Now().Add(time.Minute * time.Duration(15)).Unix()
	deadline := big.NewInt(d)

	// single trade input
	// single-hop exact input
	r, err := entities.NewRoute([]*entities.Pool{pool}, wip, usdc)
	if err != nil {
		log.Fatal(err)
	}

	swapValue := helper.FloatStringToBigInt("0.1", 18)
	trade, err := entities.FromRoute(r, coreEntities.FromRawAmount(wip, swapValue), coreEntities.ExactInput)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%v %v\n", trade.Swaps[0].InputAmount.Quotient(), trade.Swaps[0].OutputAmount.Wrapped().Quotient())
	params, err := periphery.SwapCallParameters([]*entities.Trade{trade}, &periphery.SwapOptions{
		SlippageTolerance: slippageTolerance,
		Recipient:         wallet.PublicKey,
		Deadline:          deadline,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("calldata = 0x%x\n", params.Value.String())

	tx, err := helper.TryTX(client, common.HexToAddress(helper.ContractV3SwapRouterV1),
		swapValue, params.Calldata, wallet)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(tx.Hash().String())

	// broadcast the tx
	tx, err = helper.SendTX(client, common.HexToAddress(helper.ContractV3SwapRouterV1),
		swapValue, params.Calldata, wallet)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("tx hash: %+v", tx)

	// wait for the tx to be mined
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("tx receipt: %+v", receipt)
}
