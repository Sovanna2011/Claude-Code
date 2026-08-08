"use strict";

/**
 * The translation bundles.
 *
 * These tests exist because of what was missed. CI checked that a translation
 * carried no key the English source lacked - an orphan, which is dead weight -
 * and never checked the other direction. Khmer and Thai sat at 95 of 580 keys
 * for months while the manifest declared both as supported locales, so anybody
 * switching language got a screen four-fifths in English. A fallback that works
 * is exactly what stops anyone noticing.
 *
 * So: both directions, and the two things that break at runtime rather than at
 * load time - a placeholder index that does not survive translation silently
 * drops an argument out of the middle of a sentence, and an escape that does
 * not survive puts a literal \n on the screen.
 */
const test = require("node:test");
const assert = require("node:assert");
const fs = require("node:fs");
const path = require("node:path");

const I18N = path.join(__dirname, "..", "webapp", "i18n");
const LANGUAGES = ["km", "th"];

/**
 * Reads a .properties bundle into an ordered list of entries.
 *
 * Deliberately not a Map: duplicate keys are a real mistake - the last one
 * silently wins - and a Map would swallow them.
 */
function read(file) {
	const entries = [];
	const text = fs.readFileSync(path.join(I18N, file), "utf8");
	text.split("\n").forEach(function (raw, i) {
		const line = raw.trim();
		if (!line || line.startsWith("#") || line.indexOf("=") === -1) {
			return;
		}
		const at = line.indexOf("=");
		entries.push({
			key: line.slice(0, at).trim(),
			value: line.slice(at + 1),
			line: i + 1
		});
	});
	return entries;
}

function byKey(entries) {
	const out = {};
	entries.forEach(function (e) { out[e.key] = e.value; });
	return out;
}

/** The {0}, {1} … a string interpolates, as a sorted set. */
function placeholders(value) {
	return (value.match(/\{\d+\}/g) || []).sort().filter(function (v, i, a) {
		return a.indexOf(v) === i;
	});
}

const source = read("i18n.properties");
const sourceByKey = byKey(source);

test("the English source has no duplicate keys", () => {
	const seen = {};
	const dupes = [];
	source.forEach(function (e) {
		if (seen[e.key]) {
			dupes.push(e.key + " (lines " + seen[e.key] + " and " + e.line + ")");
		}
		seen[e.key] = e.line;
	});
	assert.deepStrictEqual(dupes, [], "a duplicate key means the later one silently wins");
});

LANGUAGES.forEach(function (lang) {
	const file = "i18n_" + lang + ".properties";
	const entries = read(file);
	const translated = byKey(entries);

	test(file + " translates every key in the English source", () => {
		const missing = source
			.map(function (e) { return e.key; })
			.filter(function (k) { return !(k in translated); });
		assert.deepStrictEqual(missing, [],
			missing.length + " keys fall back to English. The manifest declares " + lang +
			" as a supported locale, which is a promise this file has to keep.");
	});

	test(file + " carries no key the English source does not have", () => {
		const orphans = entries
			.map(function (e) { return e.key; })
			.filter(function (k) { return !(k in sourceByKey); });
		assert.deepStrictEqual(orphans, [],
			"an orphan key is dead weight, and usually a typo of a real one");
	});

	test(file + " has no duplicate keys", () => {
		const seen = {};
		const dupes = [];
		entries.forEach(function (e) {
			if (seen[e.key]) { dupes.push(e.key); }
			seen[e.key] = true;
		});
		assert.deepStrictEqual(dupes, []);
	});

	test(file + " keeps every placeholder the English string interpolates", () => {
		const wrong = [];
		Object.keys(sourceByKey).forEach(function (key) {
			if (!(key in translated)) { return; }
			const want = placeholders(sourceByKey[key]);
			const got = placeholders(translated[key]);
			if (want.join(",") !== got.join(",")) {
				wrong.push(key + ": English has " + JSON.stringify(want) +
					", " + lang + " has " + JSON.stringify(got));
			}
		});
		assert.deepStrictEqual(wrong, [],
			"a dropped placeholder loses an argument out of the middle of a sentence, " +
			"and an added one renders as literal braces");
	});

	test(file + " keeps the line breaks the English string has", () => {
		const wrong = [];
		Object.keys(sourceByKey).forEach(function (key) {
			if (!(key in translated)) { return; }
			const want = (sourceByKey[key].match(/\\n/g) || []).length;
			const got = (translated[key].match(/\\n/g) || []).length;
			if (want !== got) {
				wrong.push(key + ": English has " + want + ", " + lang + " has " + got);
			}
		});
		assert.deepStrictEqual(wrong, []);
	});

	test(file + " has no empty translations", () => {
		const blank = entries
			.filter(function (e) { return e.value.trim() === ""; })
			.map(function (e) { return e.key; });
		assert.deepStrictEqual(blank, [],
			"an empty value renders as an empty label, which is worse than falling " +
			"back to English");
	});
});

test("the manifest declares exactly the locales that have a bundle", () => {
	const manifest = JSON.parse(
		fs.readFileSync(path.join(__dirname, "..", "webapp", "manifest.json"), "utf8"));
	const declared = manifest["sap.ui5"].models.i18n.settings.supportedLocales
		.filter(function (l) { return l !== ""; })
		.sort();
	assert.deepStrictEqual(declared, LANGUAGES.slice().sort(),
		"a locale declared with no bundle serves English under a language name; " +
		"a bundle with no locale declared is never loaded at all");
});

/**
 * Every {i18n>key} a view or fragment asks for has to exist.
 *
 * A missing key is not an error at runtime: SAPUI5 renders the key itself, so
 * a button reads "supplyGenrate" and the page still works. That is the kind of
 * defect that ships, which is why it is checked here rather than left to
 * somebody noticing on screen.
 */
test("every i18n key a view asks for exists in the English source", () => {
	const VIEWS = path.join(__dirname, "..", "webapp", "view");
	const files = [];
	(function walk(dir) {
		fs.readdirSync(dir, { withFileTypes: true }).forEach(function (entry) {
			const full = path.join(dir, entry.name);
			if (entry.isDirectory()) {
				walk(full);
			} else if (entry.name.endsWith(".xml")) {
				files.push(full);
			}
		});
	}(VIEWS));
	assert.ok(files.length > 0, "no views were found to check");

	const missing = [];
	files.forEach(function (file) {
		const text = fs.readFileSync(file, "utf8");
		const seen = {};
		(text.match(/i18n>[A-Za-z0-9_.]+/g) || []).forEach(function (match) {
			const key = match.slice("i18n>".length);
			if (seen[key] || Object.prototype.hasOwnProperty.call(sourceByKey, key)) {
				return;
			}
			seen[key] = true;
			missing.push(path.relative(VIEWS, file) + ": " + key);
		});
	});
	assert.deepStrictEqual(missing, [],
		"views reference i18n keys that do not exist:\n" + missing.join("\n"));
});
