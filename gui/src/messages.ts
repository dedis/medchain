import { Message, Properties } from 'protobufjs/light'
import { addJSON, EMPTY_BUFFER, registerMessage } from '@dedis/cothority/protobuf/index'
import { InstanceID } from '@dedis/cothority/byzcoin/index'
import models from './protobuf/models.json'
import { Roster, ServerIdentity, ServiceIdentity } from '@dedis/cothority/network/index'

/**
 * GetDeferredIDs
 */
export class GetDeferredIDs extends Message<GetDeferredIDs> {
  static register () {
    registerMessage('admin.GetDeferredIDs', GetDeferredIDs)
  }
}

/**
 * Status response message
 */
export class GetDeferredIDsReply extends Message<GetDeferredIDsReply> {
  static register () {
    registerMessage('admin.GetDeferredIDsReply', GetDeferredIDsReply)
  }

    readonly ids: InstanceID[];

    constructor (props?: Properties<GetDeferredIDsReply>) {
      super(props)

      this.ids = this.ids || []

      /* Protobuf aliases */

      Object.defineProperty(this, 'instanceids', {
        get (): InstanceID[] {
          return this.ids
        },
        set (value: InstanceID[]) {
          this.ids = value
        }
      })
    }
}

/**
 * DeferredID
 */
export class ShareDeferredID extends Message<ShareDeferredID> {
  static register () {
    registerMessage('admin.ShareDeferredID', ShareDeferredID)
  }

  readonly instID: InstanceID;
  readonly r: Roster;

  constructor (props?: Properties<ShareDeferredID>) {
    super(props)

    this.instID = this.instID || EMPTY_BUFFER

    /* Protobuf aliases */

    Object.defineProperty(this, 'id', {
      get (): InstanceID {
        return this.instID
      },
      set (value: InstanceID) {
        this.instID = value
      }
    })

    Object.defineProperty(this, 'roster', {
      get (): Roster {
        return this.r
      },
      set (value: Roster) {
        this.r = value
      }
    })
  }
}

/**
 * DeferredIDReply
 */
export class ShareDeferredIDReply extends Message<ShareDeferredIDReply> {
  static register () {
    registerMessage('admin.ShareDeferredIDReply', ShareDeferredIDReply)
  }

    readonly oK: boolean;

    constructor (props?: Properties<ShareDeferredIDReply>) {
      super(props)

      /* Protobuf aliases */

      Object.defineProperty(this, 'ok', {
        get (): boolean {
          return this.oK
        },
        set (value: boolean) {
          this.oK = value
        }
      })
    }
}

addJSON(models)

GetDeferredIDs.register()
GetDeferredIDsReply.register()
ShareDeferredID.register()
ShareDeferredIDReply.register()
