package medChainUtils

import (
	"io/ioutil"
	"github.com/dedis/kyber/util/key"
	"github.com/dedis/cothority"
	"strconv"
	"github.com/dedis/cothority/omniledger/darc"
	"encoding/base64"
)

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func InitKeys(numKeys int, directory string) {
	for i := 0; i < numKeys; i++ {
		temp := darc.NewSignerEd25519(nil, nil)

		private, _ := temp.GetPrivate()
		privateInBytes, _ := private.MarshalBinary()
		public := temp.Identity().Ed25519.Point
		publicInBytes, _ := public.MarshalBinary()

		err := ioutil.WriteFile(directory + "/" + strconv.Itoa(i) + "_private", []byte(base64.StdEncoding.EncodeToString(privateInBytes)), 0644)
		Check(err)
		err = ioutil.WriteFile(directory + "/" + strconv.Itoa(i) + "_public", []byte(base64.StdEncoding.EncodeToString(publicInBytes)), 0644)
		Check(err)

		kp := key.NewKeyPair(cothority.Suite)
		err = kp.Public.UnmarshalBinary(publicInBytes)
		Check(err)
		err = kp.Private.UnmarshalBinary(privateInBytes)
		Check(err)
	}
}

func LoadIdentityEd25519(pathToPublic string) darc.Identity {
	dat, err := ioutil.ReadFile(pathToPublic)
	Check(err)
	return loadIdentityEd25519FromBytes(dat)
}

func loadIdentityEd25519FromBytes(publicBytes []byte) darc.Identity {
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


func loadSignerEd25519FromBytes(publicBytes []byte, privateBytes []byte) darc.Signer {
	kp := key.NewKeyPair(cothority.Suite)
	bin, err := base64.StdEncoding.DecodeString(string(privateBytes))
	Check(err)
	err = kp.Private.UnmarshalBinary(bin)
	Check(err)
	return darc.Signer{Ed25519: &darc.SignerEd25519{
		Point:  loadIdentityEd25519FromBytes(publicBytes).Ed25519.Point,
		Secret: kp.Private,
	}}
}