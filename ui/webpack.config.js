var HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = {
	entry: './app/js/app',
	output: {
		path: __dirname + '/public',
		filename: 'bundle.js'
	},
	module: {
		loaders: [
			{test: /\.es6$/, loader: 'babel-loader'},
			{test: /\.less$/, loader: 'style-loader!css-loader!autoprefixer-loader!less-loader'},
			{test: /\.(woff|woff2|ttf|eot|svg)$/, loader: 'file-loader'},
			{test: /\.(png|jpeg|jpg)$/, loader: 'file-loader'}
		]
	},
	resolve: {
		extensions: ['', '.js', '.json', '.es6']
	},
	plugins: [new HtmlWebpackPlugin({
		title: 'INCR.IT - Dashboard',
		template: 'app/index.html'
	})]
};

