import './stylesheets/style.scss'
import { WebSocketConnection, Roster } from '@dedis/cothority/network/index'
import { ShareDeferredID, ShareDeferredIDReply, GetDeferredIDs, GetDeferredIDsReply } from './messages'
import { getRosterStr } from './roster'
import { Argument, ByzCoinRPC, ClientTransaction, Instruction } from '@dedis/cothority/byzcoin'
import { SignerEd25519 } from '@dedis/cothority/darc'

/**
 * sayHi is the entry point.
 */
export function sayHi () {
  console.log('hi')

  document.getElementById('getIDs').addEventListener('click', getIDs)
  document.getElementById('setID').addEventListener('click', setID)
  document.getElementById('spawn-project').addEventListener('click', function () {
    spawnProject().catch(
      (e) => {
        document.getElementById('spawn-result').innerHTML = e
      }
    )
  })
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

  const connection = new WebSocketConnection('ws://127.0.0.1:7771', 'ShareID')
  connection.send(new ShareDeferredID({ instID: hex2Bytes(input.value), r: roster }), ShareDeferredIDReply).then(
    (e) => {
      document.getElementById('setID-content').innerHTML = 'value set'
    },
    (e) => {
      document.getElementById('setID-content').innerHTML = e
    }
  )
}

async function spawnProject () {
  const darcEl = document.getElementById('darc-id') as HTMLFormElement
  const signerEl = document.getElementById('signer-private') as HTMLFormElement
  const byzcoinEl = document.getElementById('byzcoin-id') as HTMLFormElement
  const nameEl = document.getElementById('project-name') as HTMLFormElement
  const descriptionEl = document.getElementById('project-description') as HTMLFormElement

  const nameArg = new Argument({ name: 'name', value: Buffer.from(nameEl.value) })
  const descriptionArg = new Argument({ name: 'description', value: Buffer.from(descriptionEl.value) })

  const instruction = Instruction.createSpawn(hex2Bytes(darcEl.value), 'project', [nameArg, descriptionArg])
  const tx = ClientTransaction.make(2, instruction)

  const sid = Buffer.from(hex2Bytes(signerEl.value))
  const signer = SignerEd25519.fromBytes(sid)

  const rosterStr = getRosterStr()
  const roster = Roster.fromTOML(rosterStr)

  const rpc = await ByzCoinRPC.fromByzcoin(roster, hex2Bytes(byzcoinEl.value))
  await tx.updateCounters(rpc, [[signer]])
  tx.signWith([[signer]])
  await rpc.sendTransactionAndWait(tx)

  document.getElementById('spawn-result').innerHTML = 'spawned project with instance id' +
   instruction.deriveId().toString('hex')
}

function hex2Bytes (hex: string): Buffer {
  if (!hex) {
    return Buffer.allocUnsafe(0)
  }

  return Buffer.from(hex, 'hex')
}
