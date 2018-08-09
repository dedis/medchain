package contracts

import (
	"errors"

	"github.com/dedis/cothority/omniledger/service"
	"strings"
	"encoding/hex"
	"github.com/dedis/cothority/omniledger/darc"
	"encoding/json"
)

// Possible query types
var QueryTypes = []string{"ObfuscatedQuery", "AggregatedQuery"}

// The contract retrieves the list of projects (and actions) associated with a particular user.
var ContractProjectListID = "ProjectList"
// Contract instances can only be spawned or deleted.
// The state of this contract is to be treated as a string with results, as in the example below.
// "darc:533d2e1ba5adac0d569874b9cfe6442a2e805105c319314af2f65a5e486eb0a5%ProjectX%ObfuscatedQuery;".
func ContractProjectList(cdb service.CollectionView, inst service.Instruction, c []service.Coin) ([]service.StateChange, []service.Coin, error) {
	switch {
	case inst.Spawn != nil:
		iid := service.InstanceID{
			DarcID: inst.InstanceID.DarcID,
			SubID:  service.NewSubID(inst.Hash()),
		}
		// Get the user-projects map
		userProjectsMapInstance, _, err := cdb.GetValues(inst.Spawn.Args.Search("userProjectsMapInstanceID")[:])
		if err != nil {
			return nil, nil, errors.New("Could not retrieve the user-projects map")
		}
		userProjectsMap := make(map[string](map[string]string))
		if err := json.Unmarshal(userProjectsMapInstance, &userProjectsMap); err != nil {
			return nil, nil, errors.New("Could not unmarshal the user-projects map")
		}
		// Use the map to retrieve the information associated with a user
		output := []string{""}
		for k := range userProjectsMap[inst.Signatures[0].Signer.String()] {
			results := strings.Split(userProjectsMap[inst.Signatures[0].Signer.String()][k], "%")

			// Identity of the user
			output = append(output, inst.Signatures[0].Signer.String())
			output = append(output, "......")

			// Project description
			output = append(output, results[0])
			output = append(output, "...")

			// Project DARC ID
			output = append(output, k)

			// Project query types
			for i := 1; i < len(results); i++ {
				output = append(output, "..." + results[i])
			}
			output = append(output, "......")
		}
		return []service.StateChange{
			service.NewStateChange(service.Create, iid, ContractProjectListID, []byte(strings.Join(output, ""))),
		}, c, nil
	case inst.Invoke != nil:
		return nil, nil, errors.New("The contract can not be invoked")
	case inst.Delete != nil:
		return service.StateChanges{
			service.NewStateChange(service.Remove, inst.InstanceID, ContractProjectListID, nil),
		}, c, nil
	}
	return nil, nil, errors.New("Didn't find any instruction")
}

// The contract authorizes a user for a particular project.
var ContractAuthGrantID = "AuthGrant"
// Contract instances can only be spawned or deleted.
func ContractAuthGrant(cdb service.CollectionView, inst service.Instruction, c []service.Coin) ([]service.StateChange, []service.Coin, error) {
	switch {
	case inst.Spawn != nil:
		iid := service.InstanceID{
			DarcID: inst.InstanceID.DarcID,
			SubID:  service.NewSubID(inst.Hash()),
		}

		// Callback to find the latest DARC
		callback := DarcCallback(&cdb)

		output := ""
		retrievedDarc, err := service.LoadDarcFromColl(cdb, service.InstanceID{inst.InstanceID.DarcID, service.SubID{}}.Slice())
		if err != nil{
			return nil, nil, errors.New("Could not find given DARC")
		}
		if err := darc.EvalExprWithSigs(retrievedDarc.Rules["spawn:AuthGrant"], callback, inst.Signatures[0]); err != nil {
			output = inst.Signatures[0].Signer.String() + " could not be authorized"
		} else {
			output = inst.Signatures[0].Signer.String() + " authorized for " + string(retrievedDarc.Description[:])
		}

		return []service.StateChange{
			service.NewStateChange(service.Create, iid, ContractProjectListID, []byte(output)),
		}, c, nil
	case inst.Invoke != nil:
		return nil, nil, errors.New("The contract can not be invoked")
	case inst.Delete != nil:
		return service.StateChanges{
			service.NewStateChange(service.Remove, inst.InstanceID, ContractProjectListID, nil),
		}, c, nil
	}
	return nil, nil, errors.New("Didn't find any instruction")
}

// The contract authorizes a user for a particular project.
var ContractCreateQueryID = "CreateQuery"
// Contract instances can only be spawned or deleted.
func ContractCreateQuery(cdb service.CollectionView, inst service.Instruction, c []service.Coin) ([]service.StateChange, []service.Coin, error) {
	switch {
	case inst.Spawn != nil:
		iid := service.InstanceID{
			DarcID: inst.InstanceID.DarcID,
			SubID:  service.NewSubID(inst.Hash()),
		}
		givenQueryType := string(inst.Spawn.Args.Search("queryType")[:])
		givenQuery := string(inst.Spawn.Args.Search("query")[:])

		// Callback to find the latest DARC
		callback := DarcCallback(&cdb)

		output := ""
		retrievedDarc, err := service.LoadDarcFromColl(cdb, service.InstanceID{inst.InstanceID.DarcID, service.SubID{}}.Slice())
		if err != nil{
			return nil, nil, errors.New("Could not find the given DARC")
		}
		if err := darc.EvalExprWithSigs(retrievedDarc.Rules[darc.Action("spawn:" + givenQueryType)], callback, inst.Signatures[0]); err != nil {
			output = inst.Signatures[0].Signer.String() + " not authorized to create this query"
		} else {
			output =  givenQuery + "......" + givenQueryType + "......" +
				inst.Signatures[0].Signer.String() + "......" + string(retrievedDarc.Description[:])
		}
		return []service.StateChange{
			service.NewStateChange(service.Create, iid, ContractProjectListID, []byte(output)),
		}, c, nil
	case inst.Invoke != nil:
		return nil, nil, errors.New("The contract can not be invoked")
	case inst.Delete != nil:
		return service.StateChanges{
			service.NewStateChange(service.Remove, inst.InstanceID, ContractProjectListID, nil),
		}, c, nil
	}
	return nil, nil, errors.New("Didn't find any instruction")
}

var ContractUserProjectsMapID = "UserProjectsMap"
func ContractUserProjectsMap(cdb service.CollectionView, inst service.Instruction, c []service.Coin) ([]service.StateChange, []service.Coin, error) {
	switch {
	case inst.Spawn != nil || inst.Invoke != nil:
		var iid service.InstanceID
		var givenUsers []string
		var output map[string](map[string]string)
		var allProjectsListInstance []byte
		var action service.StateAction

		if inst.Spawn != nil {
			action = service.Create

			iid = service.InstanceID{
				DarcID: inst.InstanceID.DarcID,
				SubID:  service.NewSubID(inst.Hash()),
			}
			givenUsers = strings.Split(string(inst.Spawn.Args.Search("users")[:]), ";")
			allProjectsListInstance, _, _ = cdb.GetValues(inst.Spawn.Args.Search("allProjectsListInstanceID"))

			// Need to create a new map
			output = make(map[string](map[string]string))
		} else {
			if inst.Invoke.Command != "update" {
				return nil, nil, errors.New("Value contract can only update")
			}
			action = service.Update

			iid = inst.InstanceID
			givenUsers = strings.Split(string(inst.Invoke.Args.Search("users")[:]), ";")
			allProjectsListInstance, _, _ = cdb.GetValues(inst.Invoke.Args.Search("allProjectsListInstanceID"))

			// Need to update an existing map
			userProjectsMapInstance, _, err := cdb.GetValues(iid.Slice())
			if err != nil {
				return nil, nil, errors.New("Could not retrieve the existing user-projects map")
			}
			output = make(map[string](map[string]string))
			if err := json.Unmarshal(userProjectsMapInstance, &output); err != nil {
				return nil, nil, errors.New("Could not unmarshal the user-projects map")
			}
			// Remove existing information for these users from the map
			for u := 0; u < len(givenUsers); u++ {
				output[givenUsers[u]] = map[string]string{}
			}
		}
		allProjects := strings.Split(string(allProjectsListInstance[:]), ";")

		// Callback to find the latest DARC, to be used later
		callback := DarcCallback(&cdb)

		// Go through each project
		for p := 0; p < len(allProjects); p++ {
			// We don't need the "darc:" part of the identity string
			darcID, err := hex.DecodeString(allProjects[p][5:])
			if err != nil{
				return nil, nil, errors.New("Could not parse one of the given project DARC strings")
			}
			retrievedDarc, err := service.LoadDarcFromColl(cdb, service.InstanceID{darcID, service.SubID{}}.Slice())
			if err != nil{
				return nil, nil, errors.New("Could not find one of the given project DARCs")
			}

			// Go through each user who's map entry is to be created / updated
			for u := 0; u < len(givenUsers); u++ {
				if output[givenUsers[u]] == nil {
					output[givenUsers[u]] = map[string]string{}
				}
				// Check if the user can even get an AuthGrant for this project
				if err := darc.EvalExpr(retrievedDarc.Rules["spawn:AuthGrant"], callback, givenUsers[u]); err != nil {
					continue
				}
				output[givenUsers[u]][allProjects[p]] =  string(retrievedDarc.Description[:])
				// Go through every possible query type
				for q := 0; q < len(QueryTypes); q++ {
					if err := darc.EvalExpr(retrievedDarc.Rules[darc.Action("spawn:" + QueryTypes[q])], callback, givenUsers[u]); err != nil {
						continue
					}
					output[givenUsers[u]][allProjects[p]] += "%" + QueryTypes[q]
				}
			}
		}
		outputByte, _ := json.Marshal(output)
		return []service.StateChange{
			service.NewStateChange(action, iid, ContractUserProjectsMapID, outputByte),
		}, c, nil
	case inst.Delete != nil:
		return service.StateChanges{
			service.NewStateChange(service.Remove, inst.InstanceID, ContractValueID, nil),
		}, c, nil
	}
	return nil, nil, errors.New("didn't find any instruction")
}

// Get the callback to find the latest DARC
func DarcCallback(cdb *service.CollectionView) func(str string, latest bool) *darc.Darc {
	return func(str string, latest bool) *darc.Darc{
		darcID, err := hex.DecodeString(str[5:])
		if err != nil{
			return nil
		}
		d, err := service.LoadDarcFromColl(*cdb, service.InstanceID{darcID, service.SubID{}}.Slice())
		if err != nil{
			return nil
		}
		return d
	}
}
