"use strict";

/**
 * The formatters.
 *
 * Two things here have already gone wrong once and are worth nailing down.
 *
 * The first is the three colour enumerations. ObjectStatus takes ValueState,
 * NumericContent takes ValueColor, and sap.ui.core.Icon takes IconColor, and
 * they overlap enough to look interchangeable and differ enough that passing
 * one for another silently drops the colour. There is a formatter per
 * enumeration, and these tests are what stops somebody consolidating them.
 *
 * The second is that quantities arrive as strings, because the backend
 * serialises exact decimals as strings rather than as floats. A formatter that
 * assumed numbers would work in every test written by hand and fail on the wire.
 */
const test = require("node:test");
const assert = require("node:assert");
const { load } = require("./amd");

const formatter = load("model/formatter.js");

test("quantities arrive as strings and must still format", () => {
	// This is what the API actually sends: a string, to keep the decimal exact.
	assert.strictEqual(formatter.tons("2300000.000"), "2,300,000.000");
	assert.strictEqual(formatter.tons0("16788.321"), "16,788");
	assert.strictEqual(formatter.percent("11.000"), "11.00 %");
	assert.strictEqual(formatter.units("4826010"), "4,826,010");
});

test("an absent figure is blank, not zero", () => {
	// A tile that reads "0 t" says the factory produced nothing. A blank one
	// says nobody has told us yet, which is a different fact.
	["", null, undefined].forEach(function (v) {
		assert.strictEqual(formatter.tons(v), "", "tons of " + JSON.stringify(v));
		assert.strictEqual(formatter.tons0(v), "");
		assert.strictEqual(formatter.percent(v), "");
		assert.strictEqual(formatter.money(v), "");
	});
	// Zero is a figure and must survive.
	assert.strictEqual(formatter.tons0("0"), "0");
});

test("money is abbreviated because a KPI tile truncates rather than shortens", () => {
	// 76,606,286 arrives in a NumericContent as "76,6", which reads as
	// seventy-six point six of something.
	assert.strictEqual(formatter.money("76606286"), "76.6m");
	assert.strictEqual(formatter.money("1500000000"), "1.50bn");
	assert.strictEqual(formatter.money("45000"), "45k");
	assert.strictEqual(formatter.money("9999"), "9,999");
	assert.strictEqual(formatter.money("-76606286"), "-76.6m");
});

test("the three colour enumerations stay three", () => {
	// ObjectStatus and ProgressIndicator: sap.ui.core.ValueState.
	assert.strictEqual(formatter.severityState("ERROR"), "Error");
	assert.strictEqual(formatter.severityState("WARNING"), "Warning");
	assert.strictEqual(formatter.severityState("SUCCESS"), "Success");
	assert.strictEqual(formatter.severityState("INFO"), "Information");

	// NumericContent and the micro charts: sap.m.ValueColor.
	assert.strictEqual(formatter.severityColor("ERROR"), "Error");
	assert.strictEqual(formatter.severityColor("WARNING"), "Critical");
	assert.strictEqual(formatter.severityColor("SUCCESS"), "Good");
	assert.strictEqual(formatter.severityColor("INFO"), "Neutral");

	// sap.ui.core.Icon: sap.ui.core.IconColor. Passing "Error" here logs an
	// error at render time and drops the colour, which is how this one was
	// found.
	assert.strictEqual(formatter.severityIconColor("ERROR"), "Negative");
	assert.strictEqual(formatter.severityIconColor("WARNING"), "Critical");
	assert.strictEqual(formatter.severityIconColor("SUCCESS"), "Positive");
	assert.strictEqual(formatter.severityIconColor("INFO"), "Neutral");

	// The three must not be the same function wearing three names.
	assert.notStrictEqual(formatter.severityState("WARNING"), formatter.severityColor("WARNING"));
	assert.notStrictEqual(formatter.severityColor("ERROR"), formatter.severityIconColor("ERROR"));
});

test("an unknown value is neutral rather than an error state", () => {
	// A status this build has not heard of is not a failure; colouring it red
	// would turn every future status into an alarm.
	["", null, undefined, "SOMETHING_NEW"].forEach(function (v) {
		assert.strictEqual(formatter.severityState(v), "Information");
		assert.strictEqual(formatter.severityColor(v), "Neutral");
		assert.strictEqual(formatter.severityIconColor(v), "Neutral");
	});
});

test("achievement is graded on the plan, not on zero", () => {
	assert.strictEqual(formatter.achievementColor("100"), "Good");
	assert.strictEqual(formatter.achievementColor("98"), "Good");
	assert.strictEqual(formatter.achievementColor("97.9"), "Critical");
	assert.strictEqual(formatter.achievementColor("90"), "Critical");
	assert.strictEqual(formatter.achievementColor("89.9"), "Error");
	// A season that has not started is not a season that is failing.
	assert.strictEqual(formatter.achievementColor("0"), "Neutral");
	assert.strictEqual(formatter.achievementColor(""), "Neutral");
});

test("a schedule slip is only a slip when it is late", () => {
	assert.strictEqual(formatter.daysColor(0), "Good");
	assert.strictEqual(formatter.daysColor(-3), "Good", "finishing early is not a problem");
	assert.strictEqual(formatter.daysColor(5), "Critical");
	assert.strictEqual(formatter.daysColor(8), "Error");
});

test("a date is rendered for a reader and stored as ISO", () => {
	assert.strictEqual(formatter.date(""), "");
	assert.strictEqual(formatter.date(null), "");
	// The stored value is ISO; what comes back must not be.
	const rendered = formatter.date("2026-12-01");
	assert.notStrictEqual(rendered, "2026-12-01");
	assert.match(rendered, /2026/);
});
