import React from 'react';
import ReactDOM from 'react-dom';

import moment from 'moment';

import injectTapEventPlugin from 'react-tap-event-plugin';
injectTapEventPlugin();

import MaterialColors from 'material-ui/lib/styles/colors';

import AppBar from 'material-ui/lib/app-bar';
import Card from 'material-ui/lib/card/card';
import CardActions from 'material-ui/lib/card/card-actions';
import CardHeader from 'material-ui/lib/card/card-header';
import CardMedia from 'material-ui/lib/card/card-media';
import CardText from 'material-ui/lib/card/card-text';
import CardTitle from 'material-ui/lib/card/card-title';
import FlatButton from 'material-ui/lib/flat-button';
import CircularProgress from 'material-ui/lib/circular-progress';
import RaisedButton from 'material-ui/lib/raised-button';
import TextField from 'material-ui/lib/text-field';

import Table from 'material-ui/lib/table/table';
import TableHeaderColumn from 'material-ui/lib/table/table-header-column';
import TableRow from 'material-ui/lib/table/table-row';
import TableHeader from 'material-ui/lib/table/table-header';
import TableRowColumn from 'material-ui/lib/table/table-row-column';
import TableBody from 'material-ui/lib/table/table-body';

import { Sparklines, SparklinesLine, SparklinesBars, SparklinesReferenceLine } from 'react-sparklines';

function xhr(url, cb) {
	var xhr = new XMLHttpRequest();
	xhr.onreadystatechange = function() {
		if (xhr.readyState === 4) {
			cb(JSON.parse(xhr.responseText));
		}
	};
	xhr.open('GET', url, true);
	xhr.send();
}

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

class IncrCards extends React.Component {
	constructor() {
		super();
		this.state = {counters: [], loading: true};
	}
	componentDidMount() {
		xhr('/api/'+this.props.ns, (list) => {
			this.setState({loading: false, counters: list});
		})
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

class IncrCard extends React.Component {
	constructor() {
		super();
		this.state = {loading: true, mode: 'month'};
	}
	componentDidMount() {
		xhr('/api/'+this.props.ns+'/' + this.props.id, this.onData.bind(this));
	}
	avg(v, c) {
		return (c == 0 ? 0 : v/c);
	}
	onData(data) {
		var now = moment(data.now);
		var modes = {
			'realtime': {step: 1, format: 'ss'},
			'day': {step: 60*60, format: 'H'},
			'month': {step: 60*60*24, format: 'D'},
			'year': {step: 60*60*24*30, format: 'MMM'},
		};
		var chartData = {};
		Object.keys(modes).forEach(k => {
			var mode = modes[k];
			chartData[k] = {
				'labels': data[k].map((_, i) => {
					return moment(now).subtract(i * mode.step, 'seconds').format(mode.format);
				}).reverse(),
				'hits': data[k].concat().reverse(),
			};
		});
		this.setState({loading: false, data: data, chartData: chartData});
		if (this.state.mode == 'realtime') {
			setTimeout(xhr.bind(this, '/api/incr/' + this.props.id, this.onData.bind(this)), 1000);
		}
	}
	setMode(mode) {
		this.setState({mode: mode});
	}
	render() {
		var data;
		var total = '';
		if (this.state.loading) {
			data = <CircularProgress />
			total = '...';
		} else {
			var data = this.state.chartData[this.state.mode];
			var total = "Total: " + this.state.data.total[0];

			var labels = [<td style={{textAlign: 'left'}}>Hits</td>];
			var hits = [
				<td style={{textAlign: 'left'}}>
					<Sparklines data={data.hits} width={200}>
						<SparklinesLine color={MaterialColors.cyan500} />
						<SparklinesReferenceLine type="mean" />
					</Sparklines>
				</td>
			];

			var nonzero = (x) => (x == 0 ? '-' : <b>{x}</b>);

			if (this.state.mode == 'day') {
				labels.push(<td>Hour</td>);
				hits.push(<td></td>);
			} else if (this.state.mode == 'month') {
				labels.push(<td>Day</td>);
				hits.push(<td></td>);
			}

			var totalHits = 0;

			for (var i = 0; i < data.labels.length; i++) {
				if (this.state.mode != 'realtime') {
					labels.push(<td style={{color: MaterialColors.grey500}}>{data.labels[i]}</td>);
					hits.push(<td>{nonzero(data.hits[i])}</td>);
				}
				totalHits += data.hits[i]
			}
			if (this.state.mode != 'realtime') {
				labels.push(<td>Mean</td>);
				hits.push(<td>{Math.round(this.avg(totalHits, data.hits.length))}</td>);
			}
			var table = [<table>
				<tbody>
					<tr>{labels}</tr>
					<tr>{hits}</tr>
				</tbody>
			</table>]
			if (this.state.mode == 'realtime') {
				table.push(<p>Over the last minute ({moment().format('LT')}) received
									 {totalHits} hits</p>);
			}
			var data = table;
		}
		return <Card style={{marginBottom: '1em'}}>
			<CardHeader
				title={this.props.name}
				subtitle={total}
				actAsExpander={true}
				showExpandableButton={true}
			/>
			<CardText expandable={true}>
				<div style={{margin: '0 1em'}}>
					{data}
				</div>
			</CardText>,
			<CardActions expandable={true}>
				<FlatButton label="Realtime" primary={this.state.mode == 'realtime'}
					onClick={this.setMode.bind(this, 'realtime')}/>
				<FlatButton label="Day" primary={this.state.mode == 'day'}
					onClick={this.setMode.bind(this, 'day')}/>
				<FlatButton label="Month" primary={this.state.mode == 'month'}
					onClick={this.setMode.bind(this, 'month')}/>
				<FlatButton label="Year" primary={this.state.mode == 'year'}
					onClick={this.setMode.bind(this, 'year')}/>
			</CardActions>
		</Card>
	}
}

ReactDOM.render(<App/>, document.getElementById('container'));
