// Libraries
import React, {PureComponent} from 'react'
import {connect} from 'react-redux'

// Components
import {ErrorHandling} from 'src/shared/decorators/errors'
import WizardTextInput from 'src/reusable_ui/components/wizard/WizardTextInput'
import WizardCheckbox from 'src/reusable_ui/components/wizard/WizardCheckbox'
import KapacitorDropdown from 'src/sources/components/KapacitorDropdown'

// Actions
import {notify as notifyAction} from 'src/shared/actions/notifications'
import * as actions from 'src/shared/actions/sources'

// Utils
import {getDeep} from 'src/utils/wrappers'

// APIs
import {createKapacitor, pingKapacitor} from 'src/shared/apis'

// Constants
import {insecureSkipVerifyText} from 'src/shared/copy/tooltipText'
import {
  notifyKapacitorCreateFailed,
  notifyKapacitorCreated,
  notifyKapacitorConnectionFailed,
} from 'src/shared/copy/notifications'
import {DEFAULT_KAPACITOR} from 'src/shared/constants'

// Types
import {Kapacitor, Source} from 'src/types'

interface Props {
  notify: typeof notifyAction
  source: Source
  setError?: (b: boolean) => void
  deleteKapacitor: actions.DeleteKapacitor
  setActiveKapacitor: actions.SetActiveKapacitor
}

interface State {
  kapacitor: Kapacitor
  exists: boolean
}

@ErrorHandling
class KapacitorStep extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = {
      kapacitor: DEFAULT_KAPACITOR,
      exists: false,
    }
  }

  public next = async () => {
    const {kapacitor} = this.state
    const {notify, source, setError} = this.props

    try {
      const {data} = await createKapacitor(source, kapacitor)
      this.setState({kapacitor: data})
      this.checkKapacitorConnection(data)
      notify(notifyKapacitorCreated())
      setError(false)
      return {status: true, payload: data}
    } catch (error) {
      console.error(error)
      notify(notifyKapacitorCreateFailed())
      setError(true)
      return {status: false, payload: null}
    }
  }

  public render() {
    const {kapacitor} = this.state
    return (
      <>
        {this.kapacitorDropdown}
        <div>
          <WizardTextInput
            value={kapacitor.url}
            label="Kapacitor URL"
            onChange={this.onChangeInput('url')}
            valueModifier={this.URLModifier}
          />
          <WizardTextInput
            value={kapacitor.name}
            label="Name"
            onChange={this.onChangeInput('name')}
          />
          <WizardTextInput
            value={kapacitor.username}
            label="Username"
            onChange={this.onChangeInput('username')}
          />
          <WizardTextInput
            value={kapacitor.password}
            label="Password"
            onChange={this.onChangeInput('password')}
          />
          {this.isHTTPS && (
            <WizardCheckbox
              isChecked={kapacitor.insecureSkipVerify}
              text={`Unsafe SSL: ${insecureSkipVerifyText}`}
              onChange={this.onChangeInput('insecureSkipVerify')}
            />
          )}
        </div>
      </>
    )
  }

  private URLModifier = (value: string): string => {
    const url = value.trim()
    if (url.startsWith('http')) {
      return url
    }
    return `http://${url}`
  }

  private onChangeInput = (key: string) => (value: string | boolean) => {
    const {setError} = this.props
    const {kapacitor} = this.state
    this.setState({kapacitor: {...kapacitor, [key]: value}})
    setError(false)
  }

  private checkKapacitorConnection = async (kapacitor: Kapacitor) => {
    try {
      await pingKapacitor(kapacitor)
      this.setState({exists: true})
    } catch (error) {
      console.error(error)
      this.setState({exists: false})
      this.props.notify(notifyKapacitorConnectionFailed())
    }
  }

  private get isHTTPS(): boolean {
    const {kapacitor} = this.state
    return getDeep<string>(kapacitor, 'url', '').startsWith('https')
  }

  private get kapacitorDropdown() {
    const {source, deleteKapacitor, setActiveKapacitor} = this.props
    if (source) {
      return (
        <div>
          <KapacitorDropdown
            source={source}
            kapacitors={source.kapacitors}
            deleteKapacitor={deleteKapacitor}
            setActiveKapacitor={setActiveKapacitor}
          />
        </div>
      )
    }

    return
  }
}

const mdtp = {
  notify: notifyAction,
}

export default connect(null, mdtp, null, {withRef: true})(KapacitorStep)
