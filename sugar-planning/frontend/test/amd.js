"use strict";

/**
 * amd loads an AMD module written for SAPUI5 into plain Node.
 *
 * The two modules under test - the chart renderer and the formatters - are
 * pure: they take values and return strings, and declare no dependencies. That
 * is what makes them testable without a browser, and it is worth keeping true.
 * A module that grows a `sap/m/...` dependency will fail to load here, which is
 * the signal to move the logic back out of it rather than to weaken this shim.
 */
const fs = require("fs");
const path = require("path");
const vm = require("vm");

/**
 * load reads a module and returns what it exports.
 *
 * language pins the locale the formatters read from the UI5 core, so a test
 * asserting "2,300,000" does not depend on the machine it runs on.
 */
function load(relativePath, language) {
	const file = path.join(__dirname, "..", "webapp", relativePath);
	const source = fs.readFileSync(file, "utf8");

	let exported;
	const sandbox = {
		sap: {
			ui: {
				// The formatters ask the core for the display language at call
				// time. That is the only piece of UI5 they touch, so it is the
				// only piece stubbed here.
				getCore: function () {
					return {
						getConfiguration: function () {
							return {
								getLanguage: function () { return language || "en-GB"; }
							};
						}
					};
				},
				define: function (dependencies, factory) {
					if (dependencies && dependencies.length) {
						throw new Error(relativePath + " declares dependencies " +
							JSON.stringify(dependencies) + "; it can no longer be tested " +
							"outside a browser");
					}
					exported = factory();
				}
			}
		},
		Math: Math,
		Date: Date,
		Number: Number,
		String: String,
		Object: Object,
		Array: Array,
		JSON: JSON,
		isNaN: isNaN,
		parseFloat: parseFloat,
		parseInt: parseInt
	};
	vm.createContext(sandbox);
	vm.runInContext(source, sandbox, { filename: file });

	if (exported === undefined) {
		throw new Error(relativePath + " did not call sap.ui.define");
	}
	return exported;
}

module.exports = { load };
