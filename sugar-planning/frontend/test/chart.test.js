"use strict";

/**
 * The chart renderers.
 *
 * A chart cannot be checked by looking at it in a screenshot for every case
 * that matters - a season with no actuals yet, a day that missed target, a
 * recovery outside its range - so what is asserted here is the geometry and the
 * decisions: which bar is red, where the line stops, what the axis is anchored
 * on. A chart drawn from wrong numbers is worse than no chart, because it is
 * believed.
 */
const test = require("node:test");
const assert = require("node:assert");
const { load } = require("./amd");

const chart = load("model/chart.js");

/** point builds one day of a target-versus-actual series. */
function point(date, target, actual) {
	const has = actual !== undefined && actual !== null;
	return {
		date: date, target: String(target), actual: has ? String(actual) : "0",
		cumTarget: String(target), cumActual: has ? String(actual) : "0",
		hasActual: has
	};
}

/** days builds a run of dates from 2026-12-01. */
function days(n) {
	const out = [];
	for (let i = 0; i < n; i++) {
		const d = new Date(Date.UTC(2026, 11, 1 + i));
		out.push(d.toISOString().slice(0, 10));
	}
	return out;
}

/** countTag counts elements of one kind in the generated SVG. */
function countTag(sSvg, sTag) {
	const m = sSvg.match(new RegExp("<" + sTag + "\\b", "g"));
	return m ? m.length : 0;
}

test("every renderer says so when it has nothing to draw", () => {
	const empty = [
		chart.cumulativeCurve([]),
		chart.dailyTrend([], []),
		chart.recoveryTrend([], 9.8, 13),
		chart.productMix([]),
		chart.channelTrend([]),
		chart.downtimePareto([]),
		chart.calendar([])
	];
	empty.forEach(function (html) {
		assert.match(html, /sugarChartEmpty/,
			"an empty chart must explain itself rather than draw empty axes");
		assert.doesNotMatch(html, /<svg/, "there is nothing to plot");
	});
});

test("the daily trend colours a day that missed its target", () => {
	const html = chart.dailyTrend([
		point("2026-12-01", 100, 80),   // short
		point("2026-12-02", 100, 120),  // ahead
		point("2026-12-03", 100)        // not yet recorded
	], ["80", "100", "100"]);

	// Two bars, not three: a day with no actual is not a day of zero output.
	assert.strictEqual(countTag(html, "rect"), 2,
		"a day with no recorded actual must not be drawn as a bar");
	assert.match(html, /fill='#BB0000'/, "the short day must be marked");
	assert.match(html, /fill='#2B7D2B'/, "the day at or above target must not be");
	assert.match(html, /2026-12-01: 80 t against 100 t/,
		"every bar must carry its own figures for a screen reader and a hover");
});

test("the daily trend stops the rolling average where the actuals stop", () => {
	const aPoints = [
		point("2026-12-01", 100, 90),
		point("2026-12-02", 100, 95),
		point("2026-12-03", 100),
		point("2026-12-04", 100)
	];
	const html = chart.dailyTrend(aPoints, ["90", "92.5", "0", "0"]);
	const rolling = html.match(/stroke='#E76500' stroke-width='2'/);
	assert.ok(rolling, "the rolling average line must be drawn");

	// Two points on the rolling path, one per recorded day. A line that ran on
	// to day four would fall to zero and read as a collapse in throughput.
	const path = html.match(/<path d='(M[^']*)' fill='none' stroke='#E76500'/);
	assert.ok(path, "the rolling path must be present");
	assert.strictEqual((path[1].match(/[ML]/g) || []).length, 2,
		"the rolling average must stop at the last recorded day");
});

test("the recovery chart anchors its axis on the range, not on zero", () => {
	const aPoints = [
		{ date: "2026-12-01", target: "11.000", actual: "10.700", hasActual: true },
		{ date: "2026-12-02", target: "11.000", actual: "9.100", hasActual: true },
		{ date: "2026-12-03", target: "11.000", actual: "0", hasActual: false }
	];
	const html = chart.recoveryTrend(aPoints, "9.800", "13.000");

	// The axis labels must sit around the operating range. An axis from zero
	// would put every label between 0 % and 13 % and flatten the line.
	const labels = html.match(/>([0-9.]+ %)</g) || [];
	assert.ok(labels.length >= 5, "the axis must be labelled: " + labels);
	const values = labels.map(function (s) { return parseFloat(s.slice(1)); });
	assert.ok(Math.min.apply(null, values) > 5,
		"the axis must not start at zero, got " + values);

	// The band is drawn, and the day below the floor is marked.
	assert.match(html, /fill-opacity='0.08'/, "the operating range must be shaded");
	assert.match(html, /2026-12-02: 9\.100 %/,
		"a day outside the range must be marked and labelled");
	assert.doesNotMatch(html, /2026-12-01: 10\.700 %/,
		"a day inside the range needs no marker");
});

test("the product mix stacks the bands instead of overlaying them", () => {
	const aDates = days(3);
	function series(code, values) {
		return {
			id: code, code: code, name: code, total: "0",
			points: aDates.map(function (d, i) {
				return point(d, 0, values[i]);
			})
		};
	}
	const html = chart.productMix([series("WHT", [100, 100, 100]), series("REF", [50, 50, 50])]);

	// Three days times two products.
	assert.strictEqual(countTag(html, "rect"), 8,
		"six bands plus two legend swatches");

	// The second band must start where the first ends, not at the axis. Reading
	// the y of the two rects for the first day: the taller band sits lower.
	const ys = (html.match(/<rect x='[^']*' y='([0-9.]+)'/g) || []).slice(0, 2);
	assert.strictEqual(ys.length, 2, "two bands on the first day");

	assert.match(html, /2026-12-01 WHT: 100 t/, "each band carries its own figure");
	assert.match(html, /2026-12-01 REF: 50 t/);
});

test("the shipment chart draws the plan dashed and the actual solid per channel", () => {
	const aDates = days(2);
	const aSeries = [{
		id: "q", code: "QUOTA", name: "Quota", total: "300",
		points: [
			{ date: aDates[0], cumTarget: "500", cumActual: "300", hasActual: true },
			{ date: aDates[1], cumTarget: "1000", cumActual: "300", hasActual: false }
		]
	}];
	const html = chart.channelTrend(aSeries);

	assert.match(html, /stroke-dasharray='6 4'/, "the plan line must be dashed");
	assert.match(html, /stroke-width='2.5'/, "the actual line must be solid and heavier");
	assert.match(html, />QUOTA</, "each channel must be named in the legend");

	// The actual path stops at the recorded day: one point, not two.
	const actual = html.match(/<path d='(M[^']*)' fill='none' stroke='#5899DA' stroke-width='2.5'/);
	assert.ok(actual, "the actual path must be drawn");
	assert.strictEqual((actual[1].match(/[ML]/g) || []).length, 1,
		"the actual line must stop where the actuals do");
});

test("the downtime Pareto ranks the bars and draws the cumulative share", () => {
	const html = chart.downtimePareto([
		{ reasonCode: "DT-BOILER", reasonName: "Boiler problem", hours: "10.000",
			eventCount: 2, sharePct: "83.333", cumulativeSharePct: "83.333" },
		{ reasonCode: "DT-POWER", reasonName: "Power failure", hours: "2.000",
			eventCount: 1, sharePct: "16.667", cumulativeSharePct: "100.000" }
	]);

	assert.match(html, /Boiler problem: 10\.00 h over 2 event\(s\)/);
	assert.match(html, /DT-BOILER/);
	// The cumulative line is what makes it a Pareto rather than a bar chart.
	assert.match(html, /stroke='#E76500' stroke-width='2'/);
	assert.match(html, />100%</, "the right axis must be labelled as a share");

	// The order in the SVG is the order given, which the service sorts.
	assert.ok(html.indexOf("DT-BOILER") < html.indexOf("DT-POWER"),
		"the worst reason must be drawn first");
});

test("the downtime Pareto gathers a long tail rather than dropping it", () => {
	const aReasons = [];
	for (let i = 0; i < 12; i++) {
		aReasons.push({
			reasonCode: "R" + i, reasonName: "Reason " + i,
			hours: String(12 - i), eventCount: 1, sharePct: "8.333",
			cumulativeSharePct: String((i + 1) * 8.333)
		});
	}
	const html = chart.downtimePareto(aReasons);

	assert.match(html, />Other</,
		"the tail must be gathered into a bar so the chart still adds up");
	assert.doesNotMatch(html, />R9</, "the tail must not also be drawn separately");
	// Eight named reasons plus the gathered tail.
	assert.strictEqual((html.match(/<text x='[0-9.]+' y='[0-9]+' text-anchor='middle'/g) || []).length, 9);
});

test("the calendar places a cell per day and shades it by achievement", () => {
	const aDates = days(10);
	const aPoints = aDates.map(function (d, i) {
		if (i === 3) {
			return point(d, 100);            // no actual recorded
		}
		if (i === 5) {
			return point(d, 0, 0);           // no crushing planned
		}
		return point(d, 100, i < 2 ? 60 : 110);
	});
	const html = chart.calendar(aPoints);

	assert.strictEqual(countTag(html, "rect"), aPoints.length + 5,
		"one cell per day plus the five key swatches");
	assert.match(html, /2026-12-04: no actual recorded/);
	assert.match(html, /2026-12-06: no crushing planned/);
	assert.match(html, /2026-12-01: 60\.0 % of target/);
	assert.match(html, /fill='#BB0000'/, "a day under 70 % of target must read as bad");
	assert.match(html, /fill='#256F3A'/, "a day at or above target must read as good");

	// 2026-12-01 is a Tuesday, so the first cell belongs in the third row.
	const first = html.match(/<rect x='34' y='(\d+)'/);
	assert.ok(first, "the first cell must be placed");
	assert.strictEqual(first[1], String(22 + 2 * 17),
		"the first cell must sit on its own weekday row");
});

test("nothing a caller passes can inject markup into the chart", () => {
	const html = chart.downtimePareto([{
		reasonCode: "<script>x</script>", reasonName: "</title><script>y</script>",
		hours: "1", eventCount: 1, sharePct: "100", cumulativeSharePct: "100"
	}]);
	assert.doesNotMatch(html, /<script>/,
		"a reason code out of the database must never become markup");
	assert.match(html, /&lt;script&gt;/);
});
