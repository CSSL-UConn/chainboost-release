package crypto

import "github.com/basedfs/log"

func Testvrf() {
	t := []byte("first test")
	VrfPubkey, VrfPrivkey := VrfKeygen()
	proof, ok := VrfPrivkey.proveBytes(t)
	if !ok {
		log.LLvl2("error while generating proof")
	}
	r,_ := VrfPubkey.verifyBytes(proof, t)
	if r == true {
		log.LLvl2("proof is approved")
	} else {
		log.LLvl2("proof is rejected")
	}
}