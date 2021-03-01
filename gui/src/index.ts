import './stylesheets/style.scss'
import { WebSocketConnection, Roster } from '@dedis/cothority/network/index'
import { ShareDeferredID, ShareDeferredIDReply, GetDeferredIDs, GetDeferredIDsReply } from './messages'
import { getRosterStr } from './roster'

/**
 * sayHi is the entry point.
 */
export function sayHi () {
  console.log('hi')

  const rosterStr = getRosterStr()
  const roster = Roster.fromTOML(rosterStr)

  const instid = hex2Bytes('9cc36071ccb902a1de7e0d21a2c176d73894b1cf88ae4cc2ba4c95cd76f474f3')
  const connection3 = new WebSocketConnection('ws://127.0.0.1:7771', 'ShareID')
  connection3.send(new ShareDeferredID({ instID: instid, r: roster }), ShareDeferredIDReply).then(
    (e) => {
      console.log('ShareID Deferred ok', e)
    },
    (e) => {
      console.log('error', e)
    }
  )

  document.getElementById('getIDs').addEventListener('click', getIDs)
  document.getElementById('setID').addEventListener('click', setID)
}

function getIDs () {
  document.getElementById('ids-content').innerHTML = ''
  const connection2 = new WebSocketConnection('ws://127.0.0.1:7771', 'ShareID')
  connection2.send(new GetDeferredIDs(), GetDeferredIDsReply).then(
    (e: GetDeferredIDsReply) => {
      const res = e.ids.map(e => e.toString('hex'))
      document.getElementById('ids-content').innerHTML = res.join('<br>')
    },
    (e) => {
      document.getElementById('ids-content').innerHTML = e
    }
  )
}

function setID () {
  const rosterStr = getRosterStr()
  const roster = Roster.fromTOML(rosterStr)

  const input = document.getElementById('setIDInout') as HTMLFormElement

  const connection3 = new WebSocketConnection('ws://127.0.0.1:7771', 'ShareID')
  connection3.send(new ShareDeferredID({ instID: hex2Bytes(input.value), r: roster }), ShareDeferredIDReply).then(
    (e) => {
      document.getElementById('setID-content').innerHTML = 'value set'
    },
    (e) => {
      document.getElementById('setID-content').innerHTML = e
    }
  )
}

function hex2Bytes (hex: string) {
  if (!hex) {
    return Buffer.allocUnsafe(0)
  }

  return Buffer.from(hex, 'hex')
}
