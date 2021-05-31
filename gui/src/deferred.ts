import { ClientTransaction } from '@dedis/cothority/byzcoin'
import { addJSON, registerMessage } from '@dedis/cothority/protobuf'
import { Message } from 'protobufjs/light'
import models from './protobuf/models.json'

export class DeferredData extends Message<DeferredData> {
  static register () {
    registerMessage('deferred.DeferredData', DeferredData)
  }

  proposedtransaction: ClientTransaction
  expireblockindex: number
  instructionhashes: Buffer[]
  maxnumexecution: number
  execresult: Buffer[]

  toString (): string {
    return `Deferred instance:
- Proposed transaction: ${this.proposedtransaction.hash().toString('hex')}
- ExpireBlockIndex: ${this.expireblockindex}
- InstructionHashes: ${this.instructionhashes}
- MaxNumExecution: ${this.maxnumexecution}
- ExecResult: ${this.execresult}`
  }
}

addJSON(models)

DeferredData.register()
