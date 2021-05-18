import './stylesheets/style.scss'
import { WebSocketConnection, Roster, WebSocketAdapter } from '@dedis/cothority/network/index'
import { getRosterStr } from './roster'
import { Argument, ByzCoinRPC, ClientTransaction, Instance, Instruction } from '@dedis/cothority/byzcoin'
import { SignerEd25519 } from '@dedis/cothority/darc'
import { CatchUpMsg, CatchUpResponse, EmptyReply, Follow, Query, QueryReply, Unfollow } from './messages'

let roster: Roster
let logElement: HTMLElement

/**
 * sayHi is the entry point.
 */
export function sayHi () {
  console.log('hi')

  roster = Roster.fromTOML(getRosterStr())

  logElement = document.getElementById('result-output')

  document.getElementById('spawn-project').addEventListener('click', function () {
    spawnProject().catch(
      (e) => {
        appendLog(e)
      }
    )
  })

  document.getElementById('spawn-query').addEventListener('click', function () {
    spawnQuery().catch(
      (e) => {
        appendLog(e)
      }
    )
  })

  document.getElementById('bypros-query').addEventListener('click', byprosQuery)
  document.getElementById('bypros-follow').addEventListener('click', byprosFollow)
  document.getElementById('bypros-unfollow').addEventListener('click', byprosUnFollow)
  document.getElementById('bypros-catchup').addEventListener('click', byprosCatchup)
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

  const rpc = await ByzCoinRPC.fromByzcoin(roster, hex2Bytes(byzcoinEl.value))
  await tx.updateCounters(rpc, [[signer]])
  tx.signWith([[signer]])
  await rpc.sendTransactionAndWait(tx)

  appendLog('spawned project with instance id ' + tx.instructions[0].deriveId().toString('hex'))
}

async function spawnQuery () {
  const signerEl = document.getElementById('signer-private') as HTMLFormElement
  const projectEl = document.getElementById('query-project-instance') as HTMLFormElement
  const useridEl = document.getElementById('query-user-id') as HTMLFormElement
  const descriptionEl = document.getElementById('query-description') as HTMLFormElement
  const queryidEl = document.getElementById('query-query-id') as HTMLFormElement
  const queryDefinition = document.getElementById('query-query-definition') as HTMLFormElement
  const byzcoinEl = document.getElementById('query-byzcoin-id') as HTMLFormElement

  const nameArg = new Argument({ name: 'userID', value: Buffer.from(useridEl.value) })
  const descriptionArg = new Argument({ name: 'description', value: Buffer.from(descriptionEl.value) })
  const queryIDArg = new Argument({ name: 'queryID', value: Buffer.from(queryidEl.value) })
  const queryDefinitionArg = new Argument({ name: 'queryDefinition', value: Buffer.from(queryDefinition.value) })

  const instruction = Instruction.createSpawn(hex2Bytes(projectEl.value), 'query',
    [nameArg, descriptionArg, queryIDArg, queryDefinitionArg])

  const tx = ClientTransaction.make(2, instruction)

  const sid = Buffer.from(hex2Bytes(signerEl.value))
  const signer = SignerEd25519.fromBytes(sid)

  const rpc = await ByzCoinRPC.fromByzcoin(roster, hex2Bytes(byzcoinEl.value))

  await tx.updateCounters(rpc, [[signer]])
  tx.signWith([[signer]])
  await rpc.sendTransactionAndWait(tx)

  appendLog('spawned query with instance id ' + tx.instructions[0].deriveId().toString('hex'))
}

async function byprosQuery () {
  const sqlInput = document.getElementById('bypros-query-textarea') as HTMLTextAreaElement
  const ws = new WebSocketConnection(roster.list[0].getWebSocketAddress(), 'ByzcoinProxy')

  const query = new Query()
  query.query = sqlInput.value

  ws.send(query, QueryReply).then(
    (reply: QueryReply) => {
      appendLog('Query result: ' + reply.result.toString())
    }
  ).catch(
    (e) => {
      appendLog('failed to query: ' + e)
    }
  )
}

function byprosFollow (e: Event) {
  const ws = new WebSocketConnection(roster.list[0].getWebSocketAddress(), 'ByzcoinProxy')

  const skipchainEl = document.getElementById('bypros-follow-skipchain') as HTMLInputElement

  const msg = new Follow()
  msg.scid = hex2Bytes(skipchainEl.value)
  msg.target = roster.list[0]

  ws.send(msg, EmptyReply).then(
    (reply: EmptyReply) => {
      appendLog('Started following ' + roster.list[0].getWebSocketAddress())
    }
  ).catch(
    (e) => {
      appendLog('failed to query: ' + e)
    }
  )
}

function byprosUnFollow (e: Event) {
  const ws = new WebSocketConnection(roster.list[0].getWebSocketAddress(), 'ByzcoinProxy')

  const msg = new Unfollow()

  ws.send(msg, EmptyReply).then(
    (reply: EmptyReply) => {
      appendLog('Stopped following ' + roster.list[0].getWebSocketAddress())
    }
  ).catch(
    (e) => {
      appendLog('failed to query: ' + e)
    }
  )
}

function byprosCatchup (e: Event) {
  const ws = new WebSocketConnection(roster.list[0].getWebSocketAddress(), 'ByzcoinProxy')

  const msg = new CatchUpMsg()

  const fromblockEl = document.getElementById('bypros-catchup-fromblock') as HTMLInputElement
  const skipchainEl = document.getElementById('bypros-catchup-skipchain') as HTMLInputElement
  const updateeveryEl = document.getElementById('bypros-catchup-update-every') as HTMLInputElement

  msg.fromblock = hex2Bytes(fromblockEl.value)
  msg.scid = hex2Bytes(skipchainEl.value)
  msg.updateevery = parseInt(updateeveryEl.value, 10)
  msg.target = roster.list[0]

  ws.sendStream<CatchUpResponse>(msg, CatchUpResponse).subscribe({
    next: ([resp, ws]) => {
      console.log(resp)
      if (resp.done) {
        appendLog('Catchup done')
      } else if (resp.err !== '') {
        appendLog('Catchup error: ' + resp.err)
      } else {
        appendLog('Catchup response: ' + resp.status.message)
      }
    },
    complete: () => {
      appendLog('Catchup done')
    },
    error: (e) => {
      appendLog('failed to catchup: ' + e)
    }
  })
}

function hex2Bytes (hex: string): Buffer {
  if (!hex) {
    return Buffer.allocUnsafe(0)
  }

  return Buffer.from(hex, 'hex')
}

function appendLog (s: any) {
  const wrapper = document.createElement('div')
  wrapper.classList.add('log-entry')
  const contentWrapper = document.createElement('pre')
  contentWrapper.append(s)
  wrapper.append(contentWrapper)

  logElement.append(wrapper)
}
