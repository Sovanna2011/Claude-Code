sap.ui.define([
	"sugarplan/controller/BaseController",
	"sugarplan/model/chart"
], function (BaseController, chart) {
	"use strict";

	/**
	 * Downtime is the stoppage log.
	 *
	 * It is separate from the maintenance calendar on purpose. A maintenance
	 * window is a decision taken in advance that shortens the crushing season;
	 * a downtime event is a record of what actually stopped, entered after the
	 * fact by the shift that lived through it. Mixing them would mean an
	 * unplanned boiler failure could quietly move the end of the campaign.
	 *
	 * The Pareto is on this page as well as the dashboard because the question
	 * "which reason costs us most" is asked while looking at the log, not only
	 * from the executive overview.
	 */
	return BaseController.extend("sugarplan.controller.Downtime", {

		onInit: function () {
			this.getView().setModel(this.getService().newModel({
				events: [],
				lines: [],
				reasons: [],
				from: "",
				to: "",
				lineId: "",
				canWrite: false,
				totalHours: "0",
				totalLost: "0",
				paretoHtml: "",
				event: this._emptyEvent()
			}), "view");
			this.getRouter().getRoute("downtime").attachPatternMatched(this._onDisplay, this);
			this.onContextRefresh(this._onDisplay);
		},

		/**
		 * _emptyEvent is a new stoppage, defaulted to today with no times.
		 *
		 * The times are left blank rather than pre-filled with "now": a
		 * supervisor entering yesterday's outage at the start of a shift would
		 * otherwise have to notice and correct two fields that already looked
		 * plausible.
		 */
		_emptyEvent: function () {
			return {
				factoryId: "", lineId: "", businessDate: "",
				startTime: "", endTime: "", planned: false,
				reasonCode: "", rootCause: "", team: "", correctiveAction: ""
			};
		},

		_onDisplay: function (oEvent) {
			var sSeasonId = this.requireSeason();
			if (!sSeasonId) {
				return;
			}
			var oModel = this.getView().getModel("view");
			oModel.setProperty("/canWrite", this.can("downtime:write"));

			// A drill-down from the dashboard arrives with the window it was
			// looking at, so the page opens on the same period rather than on a
			// default the reader then has to reproduce.
			var oQuery = (oEvent && oEvent.getParameter("arguments")
				&& oEvent.getParameter("arguments")["?query"]) || {};
			if (oQuery.from) {
				oModel.setProperty("/from", oQuery.from);
			}
			if (oQuery.to) {
				oModel.setProperty("/to", oQuery.to);
			}
			if (oQuery.lineId) {
				oModel.setProperty("/lineId", oQuery.lineId);
			}

			var that = this;
			// A drill-down carries its own filter, so a saved default must not
			// then overwrite it: somebody who clicked through to a period meant
			// that period.
			var bFromDrillDown = !!(oQuery.from || oQuery.to || oQuery.lineId);
			this.initVariants("downtime", {
				collect: function () {
					return {
						from: oModel.getProperty("/from"),
						to: oModel.getProperty("/to"),
						lineId: oModel.getProperty("/lineId")
					};
				},
				apply: function (oPayload) {
					oModel.setProperty("/from", oPayload.from || "");
					oModel.setProperty("/to", oPayload.to || "");
					oModel.setProperty("/lineId", oPayload.lineId || "");
					that._load();
				}
			}, bFromDrillDown).catch(function (oProblem) {
				// A variant list that cannot be read is not a reason to refuse
				// the page: the filters still work without saved views.
				that.showError(oProblem);
			});

			this._load();
		},

		_load: function () {
			var oModel = this.getView().getModel("view");
			var oService = this.getService();
			var that = this;
			this.setBusy(true);

			Promise.all([
				oService.listDowntime({
					factoryId: this._factoryId(),
					from: oModel.getProperty("/from") || undefined,
					to: oModel.getProperty("/to") || undefined,
					lineId: oModel.getProperty("/lineId") || undefined
				}),
				oService.listMaster("production-lines", { active: "true" }),
				oService.listMaster("reason-codes", { active: "true" })
			]).then(function (aResults) {
				var aEvents = aResults[0].value || [];
				var aLines = aResults[1].value || [];
				// Only the reasons that can explain a stoppage. Offering a
				// rework or stock-correction code here would be offering a
				// choice the server is going to refuse.
				var aReasons = (aResults[2].value || []).filter(function (oReason) {
					return oReason.category === "DOWNTIME";
				});

				oModel.setProperty("/events", that._name(aEvents, aLines, aReasons));
				oModel.setProperty("/lines", aLines);
				oModel.setProperty("/reasons", aReasons);
				that._summarise(aEvents, aLines, aReasons);
				that.setBusy(false);
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/**
		 * _name resolves the identifiers on each event into something readable.
		 *
		 * The event carries a line id and a reason code because that is what it
		 * means; a table of uuids is not a log anybody can use. A code with no
		 * master record still shows as the code, so a reason somebody has since
		 * deactivated reads as something rather than as a blank.
		 */
		_name: function (aEvents, aLines, aReasons) {
			var mLines = {};
			aLines.forEach(function (oLine) {
				mLines[oLine.id] = oLine.code + " " + oLine.name;
			});
			var mReasons = {};
			aReasons.forEach(function (oReason) {
				mReasons[oReason.code] = oReason.code + " " + oReason.name;
			});
			return aEvents.map(function (oEvent) {
				var oCopy = Object.assign({}, oEvent);
				oCopy.lineName = mLines[oEvent.lineId] || "";
				oCopy.reasonName = mReasons[oEvent.reasonCode] || oEvent.reasonCode || "";
				return oCopy;
			});
		},

		/**
		 * _summarise totals the listed events and ranks their reasons.
		 *
		 * The ranking is calculated here rather than fetched because it has to
		 * describe the filtered list on screen. The dashboard's Pareto covers
		 * the whole season and is a different question; the two use the same
		 * renderer so they read the same way.
		 */
		_summarise: function (aEvents, aLines, aReasons) {
			var mRated = {};
			aLines.forEach(function (oLine) {
				mRated[oLine.id] = parseFloat(oLine.ratedTph) || 0;
			});
			var mNamed = {};
			aReasons.forEach(function (oReason) {
				mNamed[oReason.code] = oReason.name;
			});

			var fHours = 0;
			var fLost = 0;
			var mByReason = {};
			aEvents.forEach(function (oEvent) {
				var fEventHours = parseFloat(oEvent.durationHours) || 0;
				var fEventLost = fEventHours * (mRated[oEvent.lineId] || 0);
				fHours += fEventHours;
				fLost += fEventLost;

				// A stoppage recorded against no reason is still lost time.
				// Folding it into another bucket would misdirect the reader.
				var sCode = oEvent.reasonCode || "(none)";
				if (!mByReason[sCode]) {
					mByReason[sCode] = {
						reasonCode: sCode, reasonName: mNamed[sCode] || "",
						hours: 0, lostTons: 0, eventCount: 0
					};
				}
				mByReason[sCode].hours += fEventHours;
				mByReason[sCode].lostTons += fEventLost;
				mByReason[sCode].eventCount += 1;
			});

			var aRanked = Object.keys(mByReason).map(function (sCode) {
				return mByReason[sCode];
			}).sort(function (a, b) {
				return b.hours - a.hours || (a.reasonCode < b.reasonCode ? -1 : 1);
			});
			var fRunning = 0;
			aRanked.forEach(function (oReason) {
				oReason.sharePct = fHours > 0 ? (oReason.hours / fHours) * 100 : 0;
				fRunning += oReason.sharePct;
				oReason.cumulativeSharePct = fRunning;
				oReason.hours = String(oReason.hours);
			});

			var oModel = this.getView().getModel("view");
			oModel.setProperty("/totalHours", String(fHours));
			oModel.setProperty("/totalLost", String(fLost));
			oModel.setProperty("/paretoHtml", chart.downtimePareto(aRanked));
		},

		_factoryId: function () {
			var oProfile = this.getService().getSessionProfile() || {};
			var aFactories = oProfile.factories || [];
			return aFactories.length === 1 ? aFactories[0] : "";
		},

		onFilterChange: function () {
			// The variant control needs to know the screen no longer matches the
			// view it was set from, so nobody saves over one thinking it already
			// held what they are looking at.
			this.onVariantChanged();
			this._load();
		},

		onNewEvent: function () {
			var oEvent = this._emptyEvent();
			oEvent.factoryId = this._factoryId();
			this.getView().getModel("view").setProperty("/event", oEvent);
			this._dialog("sugarplan.view.fragment.DowntimeDialog").then(function (oDialog) {
				oDialog.open();
			});
		},

		onCancelEvent: function () {
			this._closeDialog("sugarplan.view.fragment.DowntimeDialog");
		},

		/**
		 * onSaveEvent posts the stoppage.
		 *
		 * The two clock times are sent as instants and the duration is left for
		 * the server to derive, so that eight hours and twenty minutes is
		 * recorded as 8.333 rather than as whatever a browser's floating point
		 * made of it.
		 */
		onSaveEvent: function () {
			var oModel = this.getView().getModel("view");
			var oEvent = oModel.getProperty("/event");
			var that = this;

			var oPayload = {
				factoryId: oEvent.factoryId || this._factoryId(),
				lineId: oEvent.lineId || undefined,
				businessDate: oEvent.businessDate,
				planned: !!oEvent.planned,
				reasonCode: oEvent.reasonCode || undefined,
				rootCause: oEvent.rootCause || undefined,
				team: oEvent.team || undefined,
				correctiveAction: oEvent.correctiveAction || undefined,
				startAt: this._instant(oEvent.businessDate, oEvent.startTime),
				endAt: this._instant(this._endDate(oEvent), oEvent.endTime)
			};

			this.setBusy(true);
			this.getService().saveDowntime(oPayload).then(function () {
				that.setBusy(false);
				that._closeDialog("sugarplan.view.fragment.DowntimeDialog");
				that.showToast(that.getText("downtimeSaved"));
				that._load();
			}).catch(function (oProblem) {
				that.showError(oProblem);
			});
		},

		/** _instant combines a date and a wall-clock time into an ISO instant. */
		_instant: function (sDate, sTime) {
			if (!sDate || !sTime) {
				return undefined;
			}
			return sDate + "T" + (sTime.length === 5 ? sTime + ":00" : sTime) + "Z";
		},

		/**
		 * _endDate is the calendar day the stoppage finished on.
		 *
		 * A night shift stops the mill at 23:00 and restarts it at 02:00; the
		 * business date is still the day the shift began. Taking the end time
		 * literally on that date would give a negative duration, which the
		 * server rejects - correctly, since it cannot tell a night shift from a
		 * typing error.
		 */
		_endDate: function (oEvent) {
			if (!oEvent.businessDate || !oEvent.endTime || !oEvent.startTime) {
				return oEvent.businessDate;
			}
			if (oEvent.endTime >= oEvent.startTime) {
				return oEvent.businessDate;
			}
			var oDate = new Date(oEvent.businessDate + "T00:00:00Z");
			oDate.setUTCDate(oDate.getUTCDate() + 1);
			return oDate.toISOString().slice(0, 10);
		},

		onNavBack: function () {
			this.navTo("launchpad");
		}
	});
});
