sap.ui.define(["sap/base/Log"], function (Log) {
	"use strict";

	/**
	 * The one place the dashboard talks to the service.
	 *
	 * It holds the bearer token, turns the API's error envelope into an Error the controllers can
	 * show, and builds the query string from a filter object — so a filter added to the bar
	 * reaches every request without each caller assembling its own URL.
	 */
	var Api = function (baseUrl) {
		this._baseUrl = baseUrl.replace(/\/+$/, "") + "/";
		this._token = window.sessionStorage.getItem("farmarea.token") || null;
	};

	Api.prototype.isSignedIn = function () {
		return !!this._token;
	};

	Api.prototype.signOut = function () {
		this._token = null;
		window.sessionStorage.removeItem("farmarea.token");
		window.sessionStorage.removeItem("farmarea.user");
	};

	Api.prototype.currentUser = function () {
		var raw = window.sessionStorage.getItem("farmarea.user");
		return raw ? JSON.parse(raw) : null;
	};

	Api.prototype.login = function (userName, password) {
		return this._request("POST", "auth/login", { userName: userName, password: password }).then(function (result) {
			this._token = result.token;
			// sessionStorage, not localStorage: the token dies with the tab rather than sitting on
			// a shared machine until it expires.
			window.sessionStorage.setItem("farmarea.token", result.token);
			window.sessionStorage.setItem("farmarea.user", JSON.stringify(result.user));
			return result.user;
		}.bind(this));
	};

	/** Turns the filter bar's state into a query string, dropping everything not set. */
	Api.prototype.query = function (filter) {
		var parts = [];
		Object.keys(filter || {}).forEach(function (key) {
			var value = filter[key];
			if (value === null || value === undefined || value === "") {
				return;
			}
			parts.push(encodeURIComponent(key) + "=" + encodeURIComponent(value));
		});
		return parts.length ? "?" + parts.join("&") : "";
	};

	Api.prototype.get = function (path, filter) {
		return this._request("GET", path + this.query(filter));
	};

	Api.prototype.post = function (path, body) {
		return this._request("POST", path, body);
	};

	Api.prototype.put = function (path, body) {
		return this._request("PUT", path, body);
	};

	Api.prototype.del = function (path) {
		return this._request("DELETE", path);
	};

	/** The absolute URL of an endpoint, for a download the browser performs itself. */
	Api.prototype.url = function (path, filter) {
		return this._baseUrl + path + this.query(filter);
	};

	Api.prototype.token = function () {
		return this._token;
	};

	/**
	 * Downloads a file through fetch rather than by navigating, because the request needs the
	 * Authorization header — a plain link cannot carry one.
	 */
	Api.prototype.download = function (path, filter, fallbackName) {
		return fetch(this.url(path, filter), {
			headers: this._token ? { Authorization: "Bearer " + this._token } : {}
		}).then(function (response) {
			if (!response.ok) {
				throw new Error("The export failed with status " + response.status + ".");
			}
			var name = fallbackName;
			var disposition = response.headers.get("Content-Disposition");
			var match = disposition && /filename="?([^"]+)"?/.exec(disposition);
			if (match) {
				name = match[1];
			}
			return response.blob().then(function (blob) {
				var url = window.URL.createObjectURL(blob);
				var anchor = document.createElement("a");
				anchor.href = url;
				anchor.download = name;
				document.body.appendChild(anchor);
				anchor.click();
				document.body.removeChild(anchor);
				window.URL.revokeObjectURL(url);
				return name;
			});
		});
	};

	Api.prototype._request = function (method, path, body) {
		var options = { method: method, headers: {} };
		if (this._token) {
			options.headers.Authorization = "Bearer " + this._token;
		}
		if (body !== undefined) {
			options.headers["Content-Type"] = "application/json";
			options.body = JSON.stringify(body);
		}

		return fetch(this._baseUrl + path, options).then(function (response) {
			if (response.status === 204) {
				return null;
			}
			return response.text().then(function (text) {
				var payload = null;
				if (text) {
					try {
						payload = JSON.parse(text);
					} catch (e) {
						Log.error("The service answered with something that is not JSON", text);
					}
				}
				if (response.ok) {
					return payload;
				}

				// The service's error envelope carries a code and, for a rejected save, the
				// fields that were wrong. Both are worth keeping on the Error.
				var error = new Error((payload && payload.message) || ("Request failed with status " + response.status));
				error.code = (payload && payload.code) || String(response.status);
				error.fields = (payload && payload.fields) || [];
				error.status = response.status;
				throw error;
			});
		});
	};

	return Api;
});
