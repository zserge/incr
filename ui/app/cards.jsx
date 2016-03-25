import React from 'react';
import ReactDOM from 'react-dom';

import CircularProgress from 'material-ui/lib/circular-progress';

import IncrCard from './card.jsx';
import xhr from './xhr.js';

export default class IncrCards extends React.Component {
	constructor() {
		super();
		this.state = {counters: [], loading: true};
	}
	componentDidMount() {
		xhr('/api/'+this.props.ns, (list) => {
			this.setState({loading: false, counters: (list || [])});
		});
	}
	render() {
		if (this.state.loading) {
			return this.renderLoading();
		} else {
			return this.renderCards();
		}
	}
	renderLoading() {
		return <div style={{width: '50px', margin: '0 auto'}}>
			<CircularProgress size={1}/>
		</div>
	}
	renderCards() {
		if (this.state.counters.length == 0) {
			return <div>No data in this namespace</div>
		}
		var cards =
			this.state.counters.map((c) => <IncrCard key={c} id={c} ns={this.props.ns} name={c} />);
		return <div>
			{cards}
		</div>
	}
}


