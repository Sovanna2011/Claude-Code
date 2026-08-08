sap.ui.define([
	"sap/ui/core/mvc/Controller",
	"sap/ui/core/Fragment",
	"sap/ui/model/json/JSONModel",
	"sap/m/MessageToast",
	"farm/area/dashboard/model/formatter"
], function (Controller, Fragment, JSONModel, MessageToast, formatter) {
	"use strict";

	// The statuses the filter offers. A blank first entry means "any", the way every other filter
	// on the estate behaves.
	var STATUSES = ["Draft", "Submitted", "UnderReview", "Approved", "Rejected", "Revised", "Closed"];

	return Controller.extend("farm.area.dashboard.controller.Projections", {
		formatter: formatter,

		onInit: function () {
			var view = this.getView();
			view.addStyleClass(this.getOwnerComponent().getContentDensityClass());

			view.setModel(new JSONModel({ items: [], totalCount: 0 }), "list");
			view.setModel(new JSONModel({ summary: "" }), "view");
			view.setModel(new JSONModel({
				seasons: [],
				seasonsOnly: [],
				farms: [],
				zones: [],
				blocks: [],
				statuses: [{ id: "", display: "" }].concat(STATUSES.map(function (s) {
					return { id: s, display: s };
				}))
			}), "lookups");

			this.getOwnerComponent().getRouter().getRoute("projections")
				.attachPatternMatched(this._onRouteMatched, this);
		},

		_api: function () {
			return this.getOwnerComponent().getApi();
		},

		_onRouteMatched: function () {
			if (!this._api().isSignedIn()) {
				this.getOwnerComponent().getRouter().navTo("login");
				return;
			}
			var user = this._api().currentUser();
			var session = this.getOwnerComponent().getModel("session");
			session.setProperty("/user", user);
			// A Report Viewer is not offered a New button that could only answer 403.
			session.setProperty("/canPlan", !!user &&
				(user.role === "Admin" || user.role === "Manager" || user.role === "Planner"));

			if (!this._loaded) {
				this._loaded = true;
				this._loadLookups();
			}
			this._reload();
		},

		_loadLookups: function () {
			var model = this.getView().getModel("lookups");
			this._api().get("lookups/seasons").then(function (seasons) {
				model.setProperty("/seasons", [{ id: "", display: "" }].concat(seasons || []));
				model.setProperty("/seasonsOnly", seasons || []);
			}).catch(this._showError.bind(this));
			// Farms and every zone and block beneath them. The two lower lists are narrowed as the
			// user chooses, but they are loaded whole so an unfiltered list still offers everything.
			["farms", "zones", "blocks"].forEach(function (kind) {
				this._api().get("lookups/" + kind).then(function (items) {
					model.setProperty("/" + kind, [{ id: "", display: "" }].concat(items || []));
				}).catch(this._showError.bind(this));
			}.bind(this));
		},

		_filter: function () {
			return {
				seasonId: this.byId("filterSeason").getSelectedKey(),
				farmId: this.byId("filterFarm").getSelectedKey(),
				zoneId: this.byId("filterZone").getSelectedKey(),
				blockId: this.byId("filterBlock").getSelectedKey(),
				projectionStatus: this.byId("filterStatus").getSelectedKey(),
				search: this.byId("filterSearch").getValue(),
				currentOnly: this.byId("filterCurrent").getSelected() ? "true" : "",
				pageSize: 200
			};
		},

		// Choosing a farm narrows the zones and clears anything below it, so the bar can never
		// describe land that does not nest — a zone in one farm with a block in another.
		onFarmFilterChanged: function () {
			var farmId = this.byId("filterFarm").getSelectedKey();
			this.byId("filterZone").setSelectedKey("");
			this.byId("filterBlock").setSelectedKey("");
			this._narrow("zones", farmId);
			this._narrow("blocks", "");
			this._reload();
		},

		onZoneFilterChanged: function () {
			var zoneId = this.byId("filterZone").getSelectedKey();
			this.byId("filterBlock").setSelectedKey("");
			this._narrow("blocks", zoneId);
			this._reload();
		},

		_narrow: function (kind, parentId) {
			return this._api().get("lookups/" + kind, parentId ? { parentId: parentId } : {})
				.then(function (items) {
					this.getView().getModel("lookups")
						.setProperty("/" + kind, [{ id: "", display: "" }].concat(items || []));
				}.bind(this)).catch(this._showError.bind(this));
		},

		onClearFilters: function () {
			["filterSeason", "filterFarm", "filterZone", "filterBlock", "filterStatus"]
				.forEach(function (id) { this.byId(id).setSelectedKey(""); }.bind(this));
			this.byId("filterSearch").setValue("");
			this.byId("filterCurrent").setSelected(true);
			this._loadLookups();
			this._reload();
		},

		_reload: function () {
			this.byId("errorStrip").setVisible(false);
			return this._api().get("projections", this._filter()).then(function (page) {
				this.getView().getModel("list").setData(page || { items: [] });
				var items = (page && page.items) || [];
				var area = items.reduce(function (sum, p) { return sum + p.totalProjectedAreaHa; }, 0);
				this.getView().getModel("view").setProperty("/summary",
					items.length + " plan(s) · " + formatter.hectares(area) + " ha projected");
			}.bind(this)).catch(this._showError.bind(this));
		},

		onFilterChanged: function () {
			this._reload();
		},

		onOpenProjection: function (event) {
			var projection = event.getSource().getBindingContext("list").getObject();
			this.getOwnerComponent().getRouter().navTo("projection", { id: projection.id });
		},

		// ---------------------------------------------------------------- new plan

		onNewProjection: function () {
			this._dialog().then(function (dialog) {
				this.byId("projectionDialogError").setVisible(false);
				this.byId("dlgProjectionNo").setValue("");
				this.byId("dlgPreparedBy").setValue(this._api().currentUser().fullName || "");
				this.byId("dlgProjectionRemark").setValue("");

				// Default the dates to the chosen season's own planting window, so the common case
				// needs no typing and the dates land inside the window the server checks.
				var seasons = this.getView().getModel("lookups").getProperty("/seasonsOnly") || [];
				var selected = this.byId("filterSeason").getSelectedKey();
				var season = seasons.filter(function (s) { return String(s.id) === String(selected); })[0] || seasons[0];
				if (season) {
					this.byId("dlgSeason").setSelectedKey(String(season.id));
				}
				this._applySeasonDefaults(season);
				dialog.open();
			}.bind(this));
		},

		// The lookup carries the season's crop year in its parentId; the window itself comes from
		// the season record, so it is fetched rather than guessed.
		_applySeasonDefaults: function (season) {
			var today = new Date().toISOString().slice(0, 10);
			this.byId("dlgProjectionDate").setValue(today);
			if (!season) {
				this.byId("dlgPlanningStart").setValue("");
				this.byId("dlgPlanningEnd").setValue("");
				return;
			}
			this._api().get("seasons/" + season.id).then(function (full) {
				this.byId("dlgPlanningStart").setValue(full.plantingWindowStart || full.startsOn || "");
				this.byId("dlgPlanningEnd").setValue(full.plantingWindowEnd || full.endsOn || "");
			}.bind(this)).catch(function () {
				// A season that cannot be read is not worth a banner here; the planner types dates.
			});
		},

		_dialog: function () {
			if (!this._dialogPromise) {
				this._dialogPromise = Fragment.load({
					id: this.getView().getId(),
					name: "farm.area.dashboard.view.ProjectionDialog",
					controller: this
				}).then(function (dialog) {
					this.getView().addDependent(dialog);
					return dialog;
				}.bind(this));
			}
			return this._dialogPromise;
		},

		onSaveProjection: function () {
			var strip = this.byId("projectionDialogError");
			strip.setVisible(false);

			var body = {
				companyId: 1,
				plantationId: 1,
				cropSeasonId: Number(this.byId("dlgSeason").getSelectedKey()),
				projectionNo: this.byId("dlgProjectionNo").getValue().trim(),
				projectionDate: this.byId("dlgProjectionDate").getValue(),
				planningStart: this.byId("dlgPlanningStart").getValue(),
				planningEnd: this.byId("dlgPlanningEnd").getValue(),
				preparedBy: this.byId("dlgPreparedBy").getValue() || null,
				remark: this.byId("dlgProjectionRemark").getValue() || null,
				version: 0
			};

			this._api().post("projections", body).then(function (saved) {
				this.byId("projectionDialog").close();
				MessageToast.show("Created " + saved.projectionNo);
				this.getOwnerComponent().getRouter().navTo("projection", { id: saved.id });
			}.bind(this)).catch(function (error) {
				// A rejected save keeps the dialog open with the server's own message on it, so the
				// planner corrects the field rather than retyping the form.
				strip.setText(messageOf(error));
				strip.setVisible(true);
			});
		},

		onCloseProjectionDialog: function () {
			this.byId("projectionDialog").close();
		},

		// ---------------------------------------------------------------- chrome

		_showError: function (error) {
			var strip = this.byId("errorStrip");
			strip.setText(messageOf(error));
			strip.setVisible(true);
		},

		onBack: function () {
			this.getOwnerComponent().getRouter().navTo("dashboard");
		},

		onSignOut: function () {
			this._api().signOut();
			this._loaded = false;
			this.getOwnerComponent().getRouter().navTo("login");
		}
	});

	function messageOf(error) {
		var message = (error && error.message) || "The request failed.";
		if (error && error.fields && error.fields.length) {
			message += " " + error.fields.map(function (f) { return f.message; }).join(" ");
		}
		return message;
	}
});
