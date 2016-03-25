import React from 'react';
import ReactDOM from 'react-dom';

import MaterialColors from 'material-ui/lib/styles/colors';

import Card from 'material-ui/lib/card/card';
import CardActions from 'material-ui/lib/card/card-actions';
import CardHeader from 'material-ui/lib/card/card-header';
import CardMedia from 'material-ui/lib/card/card-media';
import CardText from 'material-ui/lib/card/card-text';
import CardTitle from 'material-ui/lib/card/card-title';
import FlatButton from 'material-ui/lib/flat-button';
import CircularProgress from 'material-ui/lib/circular-progress';

import { Sparklines, SparklinesLine, SparklinesBars, SparklinesReferenceLine } from 'react-sparklines';

import moment from 'moment';

import xhr from './xhr.js';

export default class IncrCard extends React.Component {
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
	renderTable() {
		var data = this.state.chartData[this.state.mode];

		var labels = [<td key='label-hits' style={{textAlign: 'left'}}>Hits</td>];
		var hits = [
			<td key='sparkline' style={{textAlign: 'left'}}>
				<Sparklines data={data.hits} width={200}>
					<SparklinesLine color={MaterialColors.cyan500} />
					<SparklinesReferenceLine type="mean" />
				</Sparklines>
			</td>
		];

		var nonzero = (x) => (x == 0 ? '-' : <b>{x}</b>);

		if (this.state.mode == 'day') {
			labels.push(<td key='label-title'>Hour</td>);
			hits.push(<td key='hits-empty'></td>);
		} else if (this.state.mode == 'month') {
			labels.push(<td key='label-title'>Day</td>);
			hits.push(<td key='hits-empty'></td>);
		}

		var totalHits = 0;

		for (var i = 0; i < data.labels.length; i++) {
			if (this.state.mode != 'realtime') {
				labels.push(<td key={'label-'+i} style={{color: MaterialColors.grey500}}>{data.labels[i]}</td>);
				hits.push(<td key={'hits-'+i}>{nonzero(data.hits[i])}</td>);
			}
			totalHits += data.hits[i]
		}
		if (this.state.mode != 'realtime') {
			labels.push(<td key='label-mean'>Mean</td>);
			hits.push(<td key='hits-mean'>{Math.round(this.avg(totalHits, data.hits.length))}</td>);
		}
		
		if (this.state.mode == 'realtime') {
			return <div>
				<table>
					<tbody>
						<tr>{labels}<td></td></tr>
						<tr>{hits}<td width={'50%'}>
								<p>Over the last minute ({moment().format('LT')}) received {totalHits} hits</p>
						</td></tr>
					</tbody>
				</table>
			</div>
		} else {
			return <table>
				<tbody>
					<tr>{labels}</tr>
					<tr>{hits}</tr>
				</tbody>
			</table>;
		}
	}
	render() {
		var data;
		var total = '';
		if (this.state.loading) {
			data = <CircularProgress />
			total = '...';
		} else {
			var total = "Total: " + this.state.data.total[0];
			data = this.renderTable();
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


