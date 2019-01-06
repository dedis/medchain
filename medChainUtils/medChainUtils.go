package medChainUtils

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/contracts"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/kyber/util/key"
	"github.com/dedis/onet/network"
)

/**
This file has many helper function for the service
**/

// Panics if the error is not nil
func Check(e error) {
	if e != nil {
		panic(e)
	}
}

// Generates pairs of public-private keys
func InitKeys(numKeys int, directory string) {
	for i := 0; i < numKeys; i++ {
		temp := darc.NewSignerEd25519(nil, nil)

		private, _ := temp.GetPrivate()
		privateInBytes, _ := private.MarshalBinary()
		public := temp.Identity().Ed25519.Point
		publicInBytes, _ := public.MarshalBinary()
		err := ioutil.WriteFile(directory+"/"+strconv.Itoa(i)+"_private_"+temp.Identity().String(), []byte(base64.StdEncoding.EncodeToString(privateInBytes)), 0644)
		Check(err)
		err = ioutil.WriteFile(directory+"/"+strconv.Itoa(i)+"_public_"+temp.Identity().String(), []byte(base64.StdEncoding.EncodeToString(publicInBytes)), 0644)
		Check(err)

		kp := key.NewKeyPair(cothority.Suite)
		err = kp.Public.UnmarshalBinary(publicInBytes)
		Check(err)
		err = kp.Private.UnmarshalBinary(privateInBytes)
		Check(err)
	}
}

// Read the public key from a file, and creates the identity object
func LoadIdentityEd25519(pathToPublic string) darc.Identity {
	dat, err := ioutil.ReadFile(pathToPublic)
	Check(err)
	return LoadIdentityEd25519FromBytes(dat)
}

//Creates the identity object from bytes of the public key
func LoadIdentityEd25519FromBytes(publicBytes []byte) darc.Identity {
	kp := key.NewKeyPair(cothority.Suite)
	bin, err := base64.StdEncoding.DecodeString(string(publicBytes[:]))
	Check(err)
	err = kp.Public.UnmarshalBinary(bin)
	Check(err)
	return darc.Identity{
		Ed25519: &darc.IdentityEd25519{
			Point: kp.Public,
		},
	}
}

//Creates the identity object from bytes of the public key
// Returns the error instead of panics
func LoadIdentityEd25519FromBytesWithErr(publicBytes []byte) (darc.Identity, error) {
	kp := key.NewKeyPair(cothority.Suite)
	bin, err := base64.StdEncoding.DecodeString(string(publicBytes[:]))
	if err != nil {
		return *new(darc.Identity), err
	}
	err = kp.Public.UnmarshalBinary(bin)
	if err != nil {
		return *new(darc.Identity), err
	}
	return darc.Identity{
		Ed25519: &darc.IdentityEd25519{
			Point: kp.Public,
		},
	}, nil
}

// Read the public and private key from files, and creates the signer object
func LoadSignerEd25519(pathToPublic string, pathToPrivate string) darc.Signer {
	dat, err := ioutil.ReadFile(pathToPrivate)
	Check(err)
	kp := key.NewKeyPair(cothority.Suite)
	bin, err := base64.StdEncoding.DecodeString(string(dat[:]))
	Check(err)
	err = kp.Private.UnmarshalBinary(bin)
	Check(err)
	return darc.Signer{Ed25519: &darc.SignerEd25519{
		Point:  LoadIdentityEd25519(pathToPublic).Ed25519.Point,
		Secret: kp.Private,
	}}
}

// Creates the signer object from bytes of the public key and bytes of the private key
func LoadSignerEd25519FromBytes(publicBytes []byte, privateBytes []byte) darc.Signer {
	kp := key.NewKeyPair(cothority.Suite)
	bin, err := base64.StdEncoding.DecodeString(string(privateBytes))
	Check(err)
	err = kp.Private.UnmarshalBinary(bin)
	Check(err)
	return darc.Signer{Ed25519: &darc.SignerEd25519{
		Point:  LoadIdentityEd25519FromBytes(publicBytes).Ed25519.Point,
		Secret: kp.Private,
	}}
}

// Creates the signer object from bytes of the public key and bytes of the private key
// Returns the error instead of panics
func LoadSignerEd25519FromBytesWithErr(publicBytes []byte, privateBytes []byte) (darc.Signer, error) {
	kp := key.NewKeyPair(cothority.Suite)
	bin, err := base64.StdEncoding.DecodeString(string(privateBytes))
	if err != nil {
		return *new(darc.Signer), err
	}
	err = kp.Private.UnmarshalBinary(bin)
	if err != nil {
		return *new(darc.Signer), err
	}
	return darc.Signer{Ed25519: &darc.SignerEd25519{
		Point:  LoadIdentityEd25519FromBytes(publicBytes).Ed25519.Point,
		Secret: kp.Private,
	}}, nil
}

// Helper function to create the transaction for a query and encodes it in base 64
func CreateQueryTransaction(projectDarc string, queryType string, query string, signer darc.Signer) string {
	// We don't need the "darc:" part from the ID, and a
	projectDarcDecoded, err := hex.DecodeString(projectDarc[5:])
	Check(err)

	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(projectDarcDecoded),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractCreateQueryID,
				Args: []service.Argument{{
					Name:  "queryType",
					Value: []byte(queryType),
				}, {
					Name:  "query",
					Value: []byte(query),
				}, {
					Name:  "currentTime",
					Value: []byte(time.Now().String()),
				}},
			},
		}},
	}

	err = ctx.Instructions[0].SignBy(projectDarcDecoded, signer)
	Check(err)
	data, err := network.Marshal(&ctx)
	Check(err)
	return base64.StdEncoding.EncodeToString(data)
}

// Helper function to create the transaction for a login and encodes it in base 64
func CreateLoginTransaction(allUsersDarc string, userProjectsMap string, signer darc.Signer) string {
	allUsersDarcBytes, err := base64.StdEncoding.DecodeString(allUsersDarc)
	Check(err)
	userProjectsMapBytes, err := base64.StdEncoding.DecodeString(userProjectsMap)
	Check(err)

	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(allUsersDarcBytes),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractProjectListID,
				Args: []service.Argument{{
					Name:  "userProjectsMapInstanceID",
					Value: userProjectsMapBytes,
				}, {
					Name:  "currentTime",
					Value: []byte(time.Now().String()),
				}},
			},
		}},
	}

	err = ctx.Instructions[0].SignBy(allUsersDarcBytes, signer)
	Check(err)
	data, err := network.Marshal(&ctx)
	Check(err)
	return base64.StdEncoding.EncodeToString(data)
}

//helper function to encode a darc id in hexadecimal
func IDToHexString(id darc.ID) string {
	return hex.EncodeToString([]byte(id))
}

//helper function to encode a darc id in base64
func IDToB64String(id darc.ID) string {
	return base64.StdEncoding.EncodeToString(id)
}

//generates an expression that is valid only if at least two of the given ids are valid
func InitAtLeastTwoExpr(ids []string) expression.Expr {
	if len(ids) <= 2 {
		return expression.InitAndExpr(ids...)
	} else {
		result := expression.InitAndExpr(ids[0], ids[1])
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				if i != 0 || j != 1 {
					new_pair := expression.InitAndExpr(ids[i], ids[j])
					result = expression.InitOrExpr("("+string(result)+")", "("+string(new_pair)+")")
				}
			}
		}
		return result
	}
}

// Check if error is nil and returns an error in the http response if not
// returns true if error was not nil
func CheckError(err error, w http.ResponseWriter, r *http.Request) bool {
	if err != nil {
		http.Error(w, err.Error(), 400)
		return true
	}
	return false
}

// creates a transaction to spawn a new darc
func createTransactionForNewDARC(baseDarc *darc.Darc, rules darc.Rules, description string) (*service.ClientTransaction, *darc.Darc, error) {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	tempDarcBuff, err := tempDarc.ToProto()
	if err != nil {
		return nil, nil, err
	}
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(baseDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: service.ContractDarcID,
				Args: []service.Argument{{
					Name:  "darc",
					Value: tempDarcBuff,
				}},
			},
		}},
	}
	return &ctx, tempDarc, nil
}

// submits a transaction that spawns a new darc
func submitSignedTransactionForNewDARC(client *service.Client, tempDarc *darc.Darc, interval time.Duration, ctx *service.ClientTransaction) (*darc.Darc, error) {
	// Commit transaction
	if _, err := client.AddTransaction(*ctx); err != nil {
		return nil, err
	}

	// Verify DARC creation before returning its reference
	instID := service.NewInstanceID(tempDarc.GetBaseID())
	pr, err := client.WaitProof(instID, interval, nil)
	if err != nil || pr.InclusionProof.Match() == false {
		fmt.Println("Error at transaction submission")
		return nil, err
	}

	return tempDarc, nil
}

/**
Helper function to create a new darc
	client is the omniledger Client
	baseDarc is the darc with the spawn:darc rule
	interval is the block interval of the chain
	rules is the list of rules of the new darc
	description is the description of the new darc
	signers are the darc.Signer objects that can validate the spawn:darc rule in the baseDarc
**/
func CreateDarc(client *service.Client, baseDarc *darc.Darc, interval time.Duration, rules darc.Rules, description string, signers ...darc.Signer) (*darc.Darc, error) {
	ctx, tempDarc, err := createTransactionForNewDARC(baseDarc, rules, description)
	if err != nil {
		fmt.Println("Error at transaction creation")
		return nil, err
	}
	if err = ctx.Instructions[0].SignBy(baseDarc.GetBaseID(), signers...); err != nil {
		fmt.Println("Error at transaction signature")
		return nil, err
	}
	return submitSignedTransactionForNewDARC(client, tempDarc, interval, ctx)
}
