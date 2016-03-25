import React from 'react';
import ReactDOM from 'react-dom';

import injectTapEventPlugin from 'react-tap-event-plugin';

import AppBar from 'material-ui/lib/app-bar';
import TextField from 'material-ui/lib/text-field';
import RaisedButton from 'material-ui/lib/raised-button';

import IncrCards from './cards.jsx';

class App extends React.Component {
	constructor() {
		super();
		this.state = {ns: ''};
		if (window.location.hash) {
			this.state.ns = window.location.hash.substring(1);
		}
		this.setValue = (e) => {
			this.setState({input: e.target.value});
		};
	}
	redirect() {
		window.location.hash = '#' + this.state.input;
		this.setState({ns: this.state.input})
	}
	render() {
		var content;
		if (this.state.ns) {
			content = <IncrCards ns={this.state.ns}/>
		} else {
			content = <div style={{margin: '0 auto'}}>
				<div>Enter namespace:</div>
				<div>
					<TextField hintText="my-namespace" onChange={this.setValue} onEnterKeyDown={this.redirect.bind(this)}/>
					<RaisedButton label="Go" style={{margin: '0 1em'}} onClick={this.redirect.bind(this)}/>
				</div>
			</div>
		}
		return <div>
			<AppBar
				showMenuIconButton={false}
				title="Dashboard"
				style={{marginBottom: '1em'}} />
			{content}
		</div>
	}
}

injectTapEventPlugin();

ReactDOM.render(<App/>, document.getElementById('container'));
