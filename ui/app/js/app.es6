import m from 'mithril';
import moment from 'moment';
import Chart from 'chart.js';
import Chartist from 'chartist';

Chart.defaults.global.animation = false;
Chart.defaults.global.responsive = true;
Chart.defaults.global.maintainAspectRatio = false;

import '../less/styles.less';
import 'style-loader!css-loader!chartist/dist/chartist.min.css';

var settings = {
	charts: [{
		title: 'Foo sum/count',
		mode: 'live',
		edit: false,
		data: [{fn: 'sum', name: 'foo'}, {fn: 'count', name: 'foo'}],
	}, {
		title: 'Bar sum/count',
		mode: 'live',
		edit: false,
		data: [{fn: 'sum', name: 'bar'}, {fn: 'count', name: 'bar'}],
	}, {
		title: 'Foo vs Bar',
		mode: 'daily',
		edit: true,
		data: [{fn: 'sum', name: 'foo'}, {fn: 'count', name: 'bar'}],
	}],
};

if (localStorage.settings) {
	settings = JSON.parse(localStorage.settings);
	console.log(settings);
}

settings.save = () => {
	localStorage.settings = JSON.stringify(settings);
}

settings.save();

var layout = {
	view: () =>
		m('section',
			m('header',
				m('h1', 'INCR.IT')),
			m('section.container',
				m.component(cards, settings)))
}

var cards = {
	controller: (settings) => ({
		settings: settings,
		add: function() {
			settings.charts.push({
				title: 'New data',
				edit: true,
				data: [],
				mode: 'live',
			})
			settings.save();
		}
	}),
	view: (c) =>
		m('ul.cards',
			settings.charts.length == 0 ?
				m('p.nodata', 'No charts here. Press "+" button to create one.') : null,
			settings.charts.map((chart, i) =>
				m('li.card',
					chart.edit ?
						m.component(chartEditor, chart, c.settings) :
						m.component(chartView, chart, c.settings))),
		  m('li',
				m('a.btn.btn-add', {onclick: c.add.bind(c)}, '+')))
}

var chartView = {
	id: 0,
	getvar: (name, fn, timeframe) => {
		if (fn == 'sum' || fn == 'count' || fn == 'unique' || fn == 'avg') {
			return m.request({method: 'GET', url: `/${name}?${timeframe}=1`})
		}
		// TODO for accumulated sum/count we need to request total as well
	},
	controller: (chart, settings) => {
		var c = {
			tid: 0,
			config: function(el, init, context) {
				if (!init) {
					el.id = `chart-id-${chartView.id++}`;
					c.line = new Chartist.Line('#' + el.id, {
						labels: ['', '', ''],
						series: [[0, 0, 0]],
					}, {
						fullWidth: true,
					});
					c.onload();
				}
			},
			label: (time, mode) => {
				time = time * 1000;
				if (mode == 'live') {
					return moment(time).format('HH:mm:ss');
				} else if (mode == 'hourly') {
					return moment(time).format('HH:mm');
				} else if (mode == 'daily') {
					return moment(time).format('DD MMM');
				} else if (mode == 'weekly') {
					return moment(time).format('DD MMM YY');
				}
			},
			sync: () => {
				m.sync(chart.data.map(data =>
					chartView.getvar(data.name, data.fn, chart.mode))).then(res => {
						c.line.update({
							labels: res[0].slice().reverse().map(x => c.label(x.time, chart.mode)),
							series: res.map((x, i) => x.slice().reverse().map(y => {
								if (chart.data[i].fn == 'sum') {
									return y.sum;
								} else if (chart.data[i].fn == 'count') {
									return y.count;
								} else if (chart.data[i].fn == 'avg') {
									return (y.count == 0 ? 0 : y.sum/y.count);
								} else if (chart.data[i].fn == 'unique') {
									return y.unique;
								}
							})),
						});
					});
			},
			mode: (m) => ({
				className: chart.mode == m ? 'active' : '',
				onclick: e => {
					e.preventDefault();
					chart.mode = m;
					settings.save();
					c.sync();
				},
			}),
			edit: () => {
				chart.edit = true;
				settings.save();
			},
			onload: () => {
				c.sync();
				c.tid = setInterval(() => {
					c.sync();
				}, 5000);
			},
			onunload: () => {
				clearInterval(c.tid);
				c.line.detach();
			},
		};
		return c;
	},
	view: (c, chart, settings) =>
		m('.card-view.flex-col',
			m('.flex-row',
				m('.chart-title.flex-1', chart.title),
				m('.chart-options.flex-row.flex-none',
					m('a.btn.btn-mode', c.mode('live'), 'live'),
					m('a.btn.btn-mode', c.mode('hourly'), 'hourly'),
					m('a.btn.btn-mode', c.mode('daily'), 'daily'),
					m('a.btn.btn-mode', c.mode('weekly'), 'weekly')),
				m('.btn.btn-edit.flex-none', {onclick: c.edit.bind(c)}, 'edit')),
			m(`.ct-chart.ct-double-octave`, {config: c.config.bind(c)}))
}

var chartEditor = {
	controller: (chart, settings) => {
		var c = {
			chart: {
				title: m.prop(chart.title),
				data: [0, 1, 2, 3].map(i => {
					console.log(chart.data[i]);
					var item = (chart.data[i] || {fn:'sum', name:''});
					return {fn: m.prop(item.fn), name: m.prop(item.name)};
				}),
			},
			option: (i, fn, text) => {
				return m('option', {
					selected: c.chart.data[i].fn() == fn,
				}, text);
			},
			save: () => {
				chart.title = c.chart.title();
				chart.data = c.chart.data.map(item => {
					if (item.name() != '') {
						return {fn: item.fn(), name: item.name()};
					} else {
						return undefined;
					}
				}).filter(n => n !== undefined);
				chart.edit = false;
				settings.save();
			},
			cancel: () => {
				chart.edit = false;
				settings.save();
			},
			move: (offset, e) => {
				e.preventDefault();
				var i = settings.charts.indexOf(chart);
				if (offset == 0) {
					if (window.confirm('Delete this chart?')) {
						settings.charts.splice(i, 1);
					}
				} else {
					var j = Math.min(Math.max((i + offset), 0), settings.charts.length-1);
					if (i != j) {
						console.log(i, j);
						var tmp = settings.charts[i];
						settings.charts[i] = settings.charts[j];
						settings.charts[j] = tmp;
					}
				}
				settings.save();
			},
		};
		return c;
	},
	view: (c, chart, settings) =>
		m('.card-editor.flex-col',
			m('.flex-row.flex-none',
				m('input.title-editor.flex-1', {
					value: c.chart.title(),
					oninput: m.withAttr('value', c.chart.title),
				}),
				m('a.btn.btn-up.flex-none', {onclick: c.move.bind(c, -1)},
					m.trust('&#9650')),
				m('a.btn.btn-down.flex-none', {onclick: c.move.bind(c, 1)},
					m.trust('&#9660')),
				m('a.btn.btn-delete.flex-none', {onclick: c.move.bind(c, 0)},
					m.trust('&#x274c'))),
			m('.double-octave',
				m('.flex-col', //{style: {width: '100%'}},
					[0, 1, 2, 3].map((i) =>
						m('.chart.flex-row.flex-1',
						 m('select', {
							 oninput: (e) => {
								 c.chart.data[i]
								 	.fn(['sum', 'count', 'avg', 'unique'][e.target.selectedIndex])
							 },
						 },
							c.option(i, 'sum', 'SUM'),
							c.option(i, 'count', 'COUNT'),
							c.option(i, 'avg', 'AVERAGE'),
							c.option(i, 'unique', 'UNIQUE')),
						 m('input.var-editor.flex-1', {
							 value: c.chart.data[i].name(),
							 oninput: m.withAttr('value', c.chart.data[i].name),
						 }))),
					m('.flex-row.flex-1', {style: {marginLeft: 'auto'}},
						m('a.btn.btn-save', {onclick: c.save.bind(c)}, 'OK'),
						m('a.btn.btn-cancel', {onclick: c.cancel.bind(c)}, 'Cancel')))))
}

m.route.mode = 'hash';
m.mount(document.body, layout);
