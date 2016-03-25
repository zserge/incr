module.exports = function(url, cb) {
	var xhr = new XMLHttpRequest();
	xhr.onreadystatechange = function() {
		if (xhr.readyState === 4) {
			cb(JSON.parse(xhr.responseText));
		}
	};
	xhr.open('GET', url, true);
	xhr.send();
};


