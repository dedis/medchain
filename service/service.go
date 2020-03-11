package service

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// Clock chooses one server from the Roster at random. It
// sends a Clock to it, which is then processed on the server side
// via the code in the service package.
//
// Clock will return the time in seconds it took to run the protocol.
func (c *Client) Clock(r *onet.Roster) (*ClockReply, error) {
	dst := r.RandomServerIdentity()
	log.Lvl4("Sending message to", dst)
	reply := &ClockReply{}
	err := c.SendProtobuf(dst, &Clock{r}, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// Count will return the number of times `Clock` has been called on this
// service-node.
func (c *Client) Count(si *network.ServerIdentity) (int, error) {
	reply := &CountReply{}
	err := c.SendProtobuf(si, &Count{}, reply)
	if err != nil {
		return -1, err
	}
	return reply.Count, nil
}
