package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

type CLI struct {
	bc *Blockchain
}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  showwallet - Show address and privete from wallet file")
	fmt.Println("  createwallet - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  printchain - Print all the blocks of the blockchain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send AMOUNT of coins from FROM address to TO")
	fmt.Println("  startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!\n")
		os.Exit(1)
	}

	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)

	showWalletCmd := flag.NewFlagSet("showwallet", flag.ExitOnError)

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")

	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")

	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")

	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")

	switch os.Args[1] {
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "showwallet":
		err := showWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		printChainCmd.Parse(os.Args[2:])
	case "h":
		cli.printUsage()
		return
	default:
		os.Exit(1)
	}

	if showWalletCmd.Parsed() {
		cli.showWallet(nodeID)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockChain(*createBlockchainAddress, nodeID)
	}

	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			os.Exit(1)
		}
		cli.startNode(nodeID, *startNodeMiner)
	}
}

func (cli *CLI) createWallet(nodeID string) {
	wallets, _ := NewWallets(nodeID)
	address := wallets.CreateWallet()
	wallets.SaveToFile(nodeID)

	fmt.Printf("Your new address: %s\n", address)
}

func (cli *CLI) showWallet(nodeID string) {
	wallets, _ := NewWallets(nodeID)

	for address, w := range wallets.Wallets {
		fmt.Printf("Your new address: %v \nprivate: %x\n", address, w.PrivateKey.D.Bytes())
		fmt.Printf("public: %s\n", hex.EncodeToString(w.PublicKey))
		fmt.Printf("public hash: %s\n\n", hex.EncodeToString(HashPubKey(w.PublicKey)))
		//fmt.Printf("public: %s\n\n", Base58Encode(append(append([]byte{0x00}, HashPubKey(w.PublicKey)...), []byte{0x00, 0x00, 0x00, 0x00}...)))
		//fmt.Printf("public: %s\n\n", Base58Encode(append(append([]byte{0x00}, []byte{0x00, 0x00, 0x00, 0x00}...), HashPubKey(w.PublicKey)...)))
	}
}

func (cli *CLI) getBalance(address string, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := NewBlockchain(nodeID)
	defer bc.db.Close()

	balance := 0
	us := UTXOSet{bc}
	//us.Reindex()
	UTXOs := us.FindUTXO(HashPubKeyFromAddress([]byte(address)))

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)

}

func (cli *CLI) createBlockChain(address string, nodeID string) {
	bc := CreateBlockchain(address, nodeID)
	defer bc.db.Close()

	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	fmt.Println("Done!")
}

func (cli *CLI) send(from, to string, amount int, nodeID string, mineNow bool) {
	if !ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}
	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()

	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	//下面创建tx的时候, 不需要使用新出的coinbaseTx.
	tx := NewUTXOTransaction(&wallet, from, to, amount, &UTXOSet)
	if mineNow {
		//发送交易的人顺便挖矿, 得到奖励.
		cbTx := NewCoinbaseTX(from, "")
		newBlock := bc.MineBlock([]*Transaction{cbTx, tx})
		UTXOSet.Update(newBlock)
	} else {
		// 直接执行send的时候, 并没有设置过nodeAddress(执行StartServer时才有设置), 所以在这里设置
		nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
		sendTx(knownNodes[0], tx)
	}

	fmt.Println("Success!")
}

func (cli *CLI) printChain(nodeID string) {
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()

	bci := bc.Iterator()

	for {
		block := bci.Next()

		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)
		pow := NewProofOfWork(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Printf("%s", tx)
		}
		fmt.Printf("\n\n")

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

func (cli *CLI) startNode(nodeID, minerAddress string) {
	fmt.Printf("Starting node %s\n", nodeID)
	if len(minerAddress) > 0 {
		if ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	StartServer(nodeID, minerAddress)
}
