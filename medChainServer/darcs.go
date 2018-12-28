package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority/omniledger/darc"
)

func GetDarcInfo(w http.ResponseWriter, r *http.Request) {
	darcVal, ok := findDarc(w, r)
	if !ok {
		return
	}
	rules_descriptions := []messages.RuleDescription{}
	for action, expr := range darcVal.Rules {
		rule_desc := messages.RuleDescription{Action: string(action), Expr: string(expr)}
		rules_descriptions = append(rules_descriptions, rule_desc)
	}
	bytes, err := darcVal.ToProto()
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := messages.DarcInfoReply{Description: string(darcVal.Description), SignExpr: string(darcVal.Rules.GetSignExpr()), Rules: rules_descriptions, Bytes: bytes}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func ListDarcUsers(w http.ResponseWriter, r *http.Request) {
	darcVal, ok := findDarc(w, r)
	if !ok {
		return
	}
	hash_map := make(map[string]bool)
	err := recursivelyFindUsersOfDarc(darcVal, &hash_map)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	users := []string{}
	for user_id, _ := range hash_map {
		users = append(users, user_id)
	}
	sort.Strings(users)
	reply := messages.ListReply{Users: users}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func findDarc(w http.ResponseWriter, r *http.Request) (*darc.Darc, bool) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return nil, false
	}
	var request messages.DarcInfoRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return nil, false
	}
	var baseId string
	if request.BaseId != "" {
		baseId = request.BaseId
	} else if request.DarcId != "" {
		var ok bool
		baseId, ok = metaData.DarcIdToBaseIdMap[request.DarcId]
		if !ok {
			medChainUtils.CheckError(errors.New("No darc with this id"), w, r)
			return nil, false
		}
	} else {
		medChainUtils.CheckError(errors.New("No darc id Nor base id was given"), w, r)
		return nil, false
	}
	darcVal, ok := metaData.BaseIdToDarcMap[baseId]
	if !ok {
		medChainUtils.CheckError(errors.New("No darc with this base id"), w, r)
		return nil, false
	}
	return darcVal, true
}

func findSignersOfDarc(listDarc *darc.Darc) ([]string, error) {
	hash_map := make(map[string]bool)
	err := recursivelyFindUsersOfDarc(listDarc, &hash_map)
	if err != nil {
		return nil, err
	}
	users := []string{}
	for user_id, _ := range hash_map {
		users = append(users, user_id)
	}
	sort.Strings(users)
	return users, nil
}

func recursivelyFindUsersOfDarc(listDarc *darc.Darc, hash_map *map[string]bool) error {
	rules := listDarc.Rules
	expr := rules.GetSignExpr()
	expr_string := string(expr)
	signer_darcs := splitAndOr(expr_string)
	for _, signer_darc := range signer_darcs {
		switch {
		case strings.HasPrefix(signer_darc, "darc:"):
			base_id, ok := metaData.DarcIdToBaseIdMap[signer_darc]
			if !ok {
				return errors.New("Unknown darc id")
			}
			subordinateDarc, ok := metaData.BaseIdToDarcMap[base_id]
			if subordinateDarc != nil {
				err := recursivelyFindUsersOfDarc(subordinateDarc, hash_map)
				if err != nil {
					return err
				}
			}
		case strings.HasPrefix(signer_darc, "ed25519:"):
			hash_map_value := *hash_map
			hash_map_value[signer_darc] = true
		case signer_darc == "":
		default:
			return errors.New("Unknown signing value")
		}
	}
	return nil
}

func splitAndOr(expr string) []string {
	result := []string{}
	or_splitted := strings.Split(expr, " | ")
	for _, substring := range or_splitted {
		and_splitted := strings.Split(substring, " & ")
		result = append(result, and_splitted...)
	}
	return result
}
