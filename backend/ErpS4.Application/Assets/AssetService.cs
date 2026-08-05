using ErpS4.Application.Posting;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;

namespace ErpS4.Application.Assets;

/// <inheritdoc />
public sealed class AssetService(
    IErpDataContext context,
    IPostingEngine postingEngine,
    INumberRangeService numberRanges,
    ITenantProvider tenantProvider,
    ICurrentUser currentUser,
    TimeProvider timeProvider,
    ILogger<AssetService> logger) : IAssetService
{
    private int TenantId => tenantProvider.TenantId;

    public async Task<AssetPostingResult> AcquireAsync(
        AssetAcquisitionRequest request,
        CancellationToken cancellationToken = default)
    {
        if (request.Amount <= 0m)
        {
            return Failed(AssetErrorCodes.AmountNotPositive,
                "An acquisition amount has to be positive.", nameof(request.Amount));
        }

        var asset = await LoadAssetAsync(
            request.CompanyCode, request.AssetNumber, request.AssetSubNumber, cancellationToken);

        if (asset is null)
        {
            return Failed(AssetErrorCodes.AssetUnknown,
                $"Asset {request.AssetNumber}-{request.AssetSubNumber} does not exist in " +
                $"{request.CompanyCode}.", nameof(request.AssetNumber));
        }

        if (asset.Asset.IsPostingBlocked || asset.Asset.Status is "Retired" or "Sold" or "Scrapped")
        {
            return Failed(AssetErrorCodes.AssetBlocked,
                $"Asset {request.AssetNumber} is {asset.Asset.Status.ToLowerInvariant()}.",
                nameof(request.AssetNumber));
        }

        var accounts = await ResolveAccountsAsync(asset, cancellationToken);
        if (accounts.Missing.Count > 0)
        {
            return new AssetPostingResult { Violations = accounts.Missing };
        }

        var valueDate = request.AssetValueDate ?? request.PostingDate;

        // Debit the asset, credit the supplier or the clearing account. Posting
        // key 70 is what tells the engine this is an asset line.
        var draft = new JournalEntryDraft(
                request.CompanyCode, "AA", request.PostingDate, request.PostingDate,
                request.CurrencyCode)
            {
                HeaderText = request.Text ?? $"Acquisition {asset.Asset.Description}",
                ReferenceDocumentNumber = request.Reference,
            }
            .AddLine(new JournalEntryDraftLine(
                "70", accounts.AcquisitionValue, new Money(request.Amount, request.CurrencyCode))
            {
                Asset = request.AssetNumber,
                AssetSubNumber = request.AssetSubNumber,
                CostCenter = asset.CostCenter,
                ProfitCenter = asset.ProfitCenter,
                Text = request.Text ?? "Asset acquisition",
            })
            .AddLine(new JournalEntryDraftLine(
                request.VendorPartnerNumber is null ? "50" : "31",
                request.OffsettingAccount,
                new Money(-request.Amount, request.CurrencyCode))
            {
                BusinessPartner = request.VendorPartnerNumber,
                BaselineDate = request.VendorPartnerNumber is null ? null : request.PostingDate,
                Text = request.Text ?? "Asset acquisition",
            });

        var posted = await postingEngine.PostAsync(
            new PostingRequest(draft, "AA", "F-90", request.IdempotencyKey), cancellationToken);

        if (!posted.IsSuccess)
        {
            return new AssetPostingResult
            {
                PostingErrors = posted.Errors,
                Violations = [
                    new RuleViolation(
                        AssetErrorCodes.PostingFailed,
                        "The acquisition document was refused; see the posting errors."),
                ],
            };
        }

        var now = timeProvider.GetUtcNow().UtcDateTime;
        var user = currentUser.UserName;

        var assetDocument = await numberRanges.NextAsync(
            "ASSET", "01", asset.Asset.CompanyCodeId, posted.FiscalYear, "AA", cancellationToken);

        var header = await LoadHeaderAsync(posted, cancellationToken);

        foreach (var area in asset.Areas)
        {
            context.Add(new AssetTransaction
            {
                TenantId = TenantId,
                CompanyCodeId = asset.Asset.CompanyCodeId,
                AssetId = asset.Asset.Id,
                FiscalYear = posted.FiscalYear,
                AssetDocumentNumber = assetDocument,
                LineItemNumber = asset.Areas.IndexOf(area) + 1,
                DepreciationAreaId = area.DepreciationAreaId,
                TransactionTypeId = accounts.AcquisitionTransactionTypeId,
                PostingDate = request.PostingDate,
                DocumentDate = request.PostingDate,
                AssetValueDate = valueDate,
                FiscalPeriod = posted.FiscalPeriod,
                CurrencyCode = request.CurrencyCode,
                TransactionAmount = request.Amount,
                AmountInLocalCurrency = request.Amount,
                AmountInAreaCurrency = request.Amount,
                PartnerBusinessPartnerId = null,
                JournalEntryHeaderId = header?.Id ?? 0,
                ReferenceDocumentNumber = posted.DocumentNumber,
                Text = request.Text ?? "Acquisition",
                IsReversed = false,
                CreatedAt = now,
                CreatedBy = user,
            });

            await UpdateValuesAsync(
                asset.Asset.Id, area.DepreciationAreaId, posted.FiscalYear, request.CurrencyCode,
                acquisitions: request.Amount, depreciation: 0m, retirements: 0m, now,
                cancellationToken);
        }

        // Capitalisation is what starts depreciation: without it the asset sits
        // on the balance sheet and never charges anything.
        if (asset.Asset.CapitalizationDate is null)
        {
            asset.Asset.CapitalizationDate = valueDate;
            asset.Asset.AcquisitionDate ??= valueDate;
            asset.Asset.Status = "Capitalized";
            asset.Asset.ModifiedAt = now;
            asset.Asset.ModifiedBy = user;
        }

        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Asset {Asset} acquired for {Amount} {Currency} on document {Document}",
            request.AssetNumber, request.Amount, request.CurrencyCode, posted.DocumentNumber);

        return new AssetPostingResult
        {
            DocumentNumber = posted.DocumentNumber,
            AssetDocumentNumber = assetDocument,
            NetBookValueAfter = await NetBookValueAsync(
                asset.Asset.Id, asset.BookAreaId, posted.FiscalYear, cancellationToken),
        };
    }

    public async Task<AssetPostingResult> RetireAsync(
        AssetRetirementRequest request,
        CancellationToken cancellationToken = default)
    {
        var asset = await LoadAssetAsync(
            request.CompanyCode, request.AssetNumber, request.AssetSubNumber, cancellationToken);

        if (asset is null)
        {
            return Failed(AssetErrorCodes.AssetUnknown,
                $"Asset {request.AssetNumber}-{request.AssetSubNumber} does not exist.",
                nameof(request.AssetNumber));
        }

        if (asset.Asset.Status is "Retired" or "Sold" or "Scrapped")
        {
            return Failed(AssetErrorCodes.AssetRetired,
                $"Asset {request.AssetNumber} is already {asset.Asset.Status.ToLowerInvariant()}.",
                nameof(request.AssetNumber));
        }

        if (asset.Asset.CapitalizationDate is null)
        {
            return Failed(AssetErrorCodes.AssetNotCapitalized,
                $"Asset {request.AssetNumber} was never capitalised, so there is nothing to retire.",
                nameof(request.AssetNumber));
        }

        var accounts = await ResolveAccountsAsync(asset, cancellationToken);
        if (accounts.Missing.Count > 0)
        {
            return new AssetPostingResult { Violations = accounts.Missing };
        }

        var year = (short)request.PostingDate.Year;
        var values = await context.Query<AssetValue>()
            .FirstOrDefaultAsync(
                v => v.TenantId == TenantId
                     && v.AssetId == asset.Asset.Id
                     && v.DepreciationAreaId == asset.BookAreaId
                     && v.FiscalYear == year,
                cancellationToken);

        var acquisitionValue = (values?.AcquisitionValueBroughtForward ?? 0m)
                               + (values?.CurrentYearAcquisitions ?? 0m);
        var accumulated = (values?.AccumulatedDepreciationBroughtForward ?? 0m)
                          + (values?.OrdinaryDepreciationPosted ?? 0m);
        var netBookValue = acquisitionValue - accumulated;

        // Sale proceeds against net book value: the difference is the gain or
        // the loss, and it has to be posted, not left in the asset account.
        var gainOrLoss = request.RevenueAmount - netBookValue;

        var draft = new JournalEntryDraft(
            request.CompanyCode, "AA", request.PostingDate, request.PostingDate,
            request.CurrencyCode)
        {
            HeaderText = request.Text ?? $"Retirement {asset.Asset.Description}",
        };

        if (request.RevenueAmount > 0m)
        {
            draft.AddLine(new JournalEntryDraftLine(
                request.CustomerPartnerNumber is null ? "40" : "01",
                request.RevenueOffsettingAccount ?? accounts.AcquisitionValue,
                new Money(request.RevenueAmount, request.CurrencyCode))
            {
                BusinessPartner = request.CustomerPartnerNumber,
                BaselineDate = request.CustomerPartnerNumber is null ? null : request.PostingDate,
                Text = "Proceeds from asset sale",
            });
        }

        if (accumulated > 0m)
        {
            draft.AddLine(new JournalEntryDraftLine(
                "40", accounts.AccumulatedDepreciation,
                new Money(accumulated, request.CurrencyCode))
            {
                Asset = request.AssetNumber,
                AssetSubNumber = request.AssetSubNumber,
                Text = "Accumulated depreciation removed",
            });
        }

        draft.AddLine(new JournalEntryDraftLine(
            "75", accounts.AcquisitionValue,
            new Money(-acquisitionValue, request.CurrencyCode))
        {
            Asset = request.AssetNumber,
            AssetSubNumber = request.AssetSubNumber,
            CostCenter = asset.CostCenter,
            ProfitCenter = asset.ProfitCenter,
            Text = "Asset retired",
        });

        if (gainOrLoss != 0m)
        {
            var isGain = gainOrLoss > 0m;
            draft.AddLine(new JournalEntryDraftLine(
                isGain ? "50" : "40",
                isGain ? accounts.RetirementGain : accounts.RetirementLoss,
                new Money(isGain ? -gainOrLoss : -gainOrLoss, request.CurrencyCode))
            {
                CostCenter = asset.CostCenter,
                ProfitCenter = asset.ProfitCenter,
                Text = isGain ? "Gain on retirement" : "Loss on retirement",
            });
        }

        var posted = await postingEngine.PostAsync(
            new PostingRequest(draft, "AA", "ABAVN", request.IdempotencyKey), cancellationToken);

        if (!posted.IsSuccess)
        {
            return new AssetPostingResult
            {
                PostingErrors = posted.Errors,
                GainOrLoss = gainOrLoss,
                Violations = [
                    new RuleViolation(
                        AssetErrorCodes.PostingFailed,
                        "The retirement document was refused; see the posting errors."),
                ],
            };
        }

        var now = timeProvider.GetUtcNow().UtcDateTime;
        asset.Asset.Status = request.RevenueAmount > 0m ? "Sold" : "Scrapped";
        asset.Asset.DeactivationDate = request.PostingDate;
        asset.Asset.ModifiedAt = now;
        asset.Asset.ModifiedBy = currentUser.UserName;

        if (values is not null)
        {
            values.CurrentYearRetirements += acquisitionValue;
            values.NetBookValue = 0m;
            values.LastUpdatedAt = now;
        }

        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Asset {Asset} retired on {Document} with {Result} of {Amount}",
            request.AssetNumber, posted.DocumentNumber,
            gainOrLoss >= 0 ? "gain" : "loss", Math.Abs(gainOrLoss));

        return new AssetPostingResult
        {
            DocumentNumber = posted.DocumentNumber,
            GainOrLoss = gainOrLoss,
            NetBookValueAfter = 0m,
        };
    }

    public async Task<DepreciationPlan> PlanDepreciationAsync(
        DepreciationRunRequest request,
        CancellationToken cancellationToken = default)
    {
        var (period, violation) = await LoadPeriodAsync(request, cancellationToken);
        if (period is null)
        {
            return new DepreciationPlan
            {
                FiscalYear = request.FiscalYear,
                FiscalPeriod = request.FiscalPeriod,
                Items = [],
                Violations = [violation!],
            };
        }

        var items = await BuildPlanAsync(request, period, cancellationToken);

        return new DepreciationPlan
        {
            FiscalYear = request.FiscalYear,
            FiscalPeriod = request.FiscalPeriod,
            Items = items,
        };
    }

    public async Task<DepreciationRunResult> RunDepreciationAsync(
        DepreciationRunRequest request,
        CancellationToken cancellationToken = default)
    {
        var (period, violation) = await LoadPeriodAsync(request, cancellationToken);
        if (period is null)
        {
            return new DepreciationRunResult
            {
                FiscalYear = request.FiscalYear,
                FiscalPeriod = request.FiscalPeriod,
                Violations = [violation!],
            };
        }

        var companyCode = await context.Query<CompanyCode>()
            .AsNoTracking()
            .FirstAsync(c => c.TenantId == TenantId && c.CompanyCodeKey == request.CompanyCode,
                cancellationToken);

        // A planned run happens once per period. Re-running needs RunType
        // Repeat, which is a deliberate act, not a double click.
        var previous = await context.Query<DepreciationRun>()
            .AnyAsync(
                r => r.TenantId == TenantId
                     && r.CompanyCodeId == companyCode.Id
                     && r.FiscalYear == request.FiscalYear
                     && r.FiscalPeriod == request.FiscalPeriod
                     && r.Status == "Completed",
                cancellationToken);

        if (previous && request.RunType == "Planned")
        {
            return new DepreciationRunResult
            {
                FiscalYear = request.FiscalYear,
                FiscalPeriod = request.FiscalPeriod,
                Violations = [
                    new RuleViolation(
                        AssetErrorCodes.RunAlreadyExecuted,
                        $"Depreciation for {request.FiscalPeriod}/{request.FiscalYear} has " +
                        "already run. Use RunType 'Repeat' to run it again."),
                ],
            };
        }

        var plan = await BuildPlanAsync(request, period, cancellationToken);
        var charges = plan.Where(p => p.Amount > 0m).ToList();

        var now = timeProvider.GetUtcNow().UtcDateTime;
        var user = currentUser.UserName;

        var run = new DepreciationRun
        {
            TenantId = TenantId,
            CompanyCodeId = companyCode.Id,
            FiscalYear = request.FiscalYear,
            FiscalPeriod = request.FiscalPeriod,
            RunType = request.RunType,
            IsTestRun = false,
            Status = "Running",
            AssetsProcessed = charges.Count,
            TotalDepreciationAmount = charges.Sum(c => c.Amount),
            ErrorCount = 0,
            StartedAt = now,
            ExecutedBy = user,
            CreatedAt = now,
            CreatedBy = user,
        };

        context.Add(run);
        await context.SaveChangesAsync(cancellationToken);

        if (charges.Count == 0)
        {
            run.Status = "Completed";
            run.CompletedAt = now;
            await context.SaveChangesAsync(cancellationToken);

            return new DepreciationRunResult
            {
                FiscalYear = request.FiscalYear,
                FiscalPeriod = request.FiscalPeriod,
                AssetsProcessed = 0,
                TotalDepreciation = 0m,
            };
        }

        var posted = await PostDepreciationAsync(
            request, companyCode, period, charges, cancellationToken);

        if (!posted.IsSuccess)
        {
            run.Status = "Failed";
            run.ErrorCount = posted.Errors.Count;
            run.CompletedAt = now;
            run.LogText = string.Join("; ", posted.Errors.Select(e => e.ToString()));
            await context.SaveChangesAsync(cancellationToken);

            return new DepreciationRunResult
            {
                FiscalYear = request.FiscalYear,
                FiscalPeriod = request.FiscalPeriod,
                PostingErrors = posted.Errors,
                Violations = [
                    new RuleViolation(
                        AssetErrorCodes.PostingFailed,
                        "The depreciation document was refused; see the posting errors."),
                ],
            };
        }

        var header = await LoadHeaderAsync(posted, cancellationToken);

        foreach (var charge in charges)
        {
            var asset = await LoadAssetAsync(
                request.CompanyCode, charge.AssetNumber, charge.AssetSubNumber, cancellationToken);

            if (asset is null)
            {
                continue;
            }

            var areaId = asset.Areas
                .First(a => a.DepreciationArea == charge.DepreciationArea).DepreciationAreaId;

            context.Add(new DepreciationPosting
            {
                TenantId = TenantId,
                DepreciationRunId = run.Id,
                AssetId = asset.Asset.Id,
                DepreciationAreaId = areaId,
                FiscalYear = request.FiscalYear,
                FiscalPeriod = request.FiscalPeriod,
                PostingDate = period.PeriodEndDate,
                CurrencyCode = companyCode.LocalCurrencyCode,
                OrdinaryDepreciationAmount = charge.Amount,
                SpecialDepreciationAmount = 0m,
                UnplannedDepreciationAmount = 0m,
                ImpairmentAmount = 0m,
                WriteUpAmount = 0m,
                TotalPostedAmount = charge.Amount,
                NetBookValueAfterPosting = charge.NetBookValueAfter,
                CostCenterId = asset.Asset.CostCenterId,
                ProfitCenterId = asset.Asset.ProfitCenterId,
                JournalEntryHeaderId = header?.Id,
                IsReversed = false,
                CreatedAt = now,
                CreatedBy = user,
            });

            await UpdateValuesAsync(
                asset.Asset.Id, areaId, request.FiscalYear, companyCode.LocalCurrencyCode,
                acquisitions: 0m, depreciation: charge.Amount, retirements: 0m, now,
                cancellationToken);
        }

        run.Status = "Completed";
        run.CompletedAt = now;
        await context.SaveChangesAsync(cancellationToken);

        logger.LogInformation(
            "Depreciation {Period}/{Year} posted {Total} over {Count} assets on {Document}",
            request.FiscalPeriod, request.FiscalYear, run.TotalDepreciationAmount,
            charges.Count, posted.DocumentNumber);

        return new DepreciationRunResult
        {
            FiscalYear = request.FiscalYear,
            FiscalPeriod = request.FiscalPeriod,
            DocumentNumber = posted.DocumentNumber,
            AssetsProcessed = charges.Count,
            TotalDepreciation = charges.Sum(c => c.Amount),
            Postings = charges,
        };
    }

    /// <summary>
    /// One document for the whole run: depreciation expense against accumulated
    /// depreciation, one pair of lines per asset so the register can be traced
    /// back line by line.
    /// </summary>
    private async Task<PostingResult> PostDepreciationAsync(
        DepreciationRunRequest request,
        CompanyCode companyCode,
        FiscalPeriod period,
        IReadOnlyList<PlannedDepreciation> charges,
        CancellationToken cancellationToken)
    {
        var draft = new JournalEntryDraft(
            request.CompanyCode, "AA", period.PeriodEndDate, period.PeriodEndDate,
            companyCode.LocalCurrencyCode)
        {
            HeaderText = $"Depreciation {request.FiscalPeriod}/{request.FiscalYear}",
        };

        foreach (var charge in charges)
        {
            var asset = await LoadAssetAsync(
                request.CompanyCode, charge.AssetNumber, charge.AssetSubNumber, cancellationToken);

            if (asset is null)
            {
                continue;
            }

            var accounts = await ResolveAccountsAsync(asset, cancellationToken);
            if (accounts.Missing.Count > 0)
            {
                continue;
            }

            draft.AddLine(new JournalEntryDraftLine(
                "40", accounts.DepreciationExpense,
                new Money(charge.Amount, companyCode.LocalCurrencyCode))
            {
                Asset = charge.AssetNumber,
                AssetSubNumber = charge.AssetSubNumber,
                CostCenter = asset.CostCenter,
                ProfitCenter = asset.ProfitCenter,
                Text = $"Depreciation {charge.AssetNumber}",
            });

            draft.AddLine(new JournalEntryDraftLine(
                "50", accounts.AccumulatedDepreciation,
                new Money(-charge.Amount, companyCode.LocalCurrencyCode))
            {
                Asset = charge.AssetNumber,
                AssetSubNumber = charge.AssetSubNumber,
                Text = $"Depreciation {charge.AssetNumber}",
            });
        }

        return await postingEngine.PostAsync(
            new PostingRequest(draft, "AA", "AFAB", request.IdempotencyKey), cancellationToken);
    }

    /// <summary>Runs the calculator over every asset in scope.</summary>
    private async Task<List<PlannedDepreciation>> BuildPlanAsync(
        DepreciationRunRequest request,
        FiscalPeriod period,
        CancellationToken cancellationToken)
    {
        var rows = await (
            from asset in context.Query<Asset>().AsNoTracking()
            join company in context.Query<CompanyCode>() on asset.CompanyCodeId equals company.Id
            join area in context.Query<AssetDepreciationArea>() on asset.Id equals area.AssetId
            join depreciationArea in context.Query<DepreciationArea>()
                on area.DepreciationAreaId equals depreciationArea.Id
            join key in context.Query<DepreciationKey>() on area.DepreciationKeyId equals key.Id
            where asset.TenantId == TenantId
                  && company.CompanyCodeKey == request.CompanyCode
                  && asset.CapitalizationDate != null
                  && asset.Status != "Retired" && asset.Status != "Sold" && asset.Status != "Scrapped"
                  && !area.IsDeactivated
                  && depreciationArea.PostsToGeneralLedger == "RealTime"
                  && (request.AssetNumber == null || asset.AssetNumber == request.AssetNumber)
            select new
            {
                asset.Id,
                asset.AssetNumber,
                asset.AssetSubNumber,
                AreaId = depreciationArea.Id,
                AreaCode = depreciationArea.DepreciationAreaCode,
                area.UsefulLifeYears,
                area.UsefulLifePeriods,
                area.DepreciationStartDate,
                area.ScrapValue,
                key.DepreciationMethod,
                key.DeclineFactor,
                key.ChangeMethodAtEnd,
                asset.CapitalizationDate,
            }).ToListAsync(cancellationToken);

        var plan = new List<PlannedDepreciation>(rows.Count);

        foreach (var row in rows)
        {
            var values = await context.Query<AssetValue>()
                .AsNoTracking()
                .FirstOrDefaultAsync(
                    v => v.TenantId == TenantId
                         && v.AssetId == row.Id
                         && v.DepreciationAreaId == row.AreaId
                         && v.FiscalYear == request.FiscalYear,
                    cancellationToken);

            var acquisition = (values?.AcquisitionValueBroughtForward ?? 0m)
                              + (values?.CurrentYearAcquisitions ?? 0m)
                              - (values?.CurrentYearRetirements ?? 0m);

            var accumulated = (values?.AccumulatedDepreciationBroughtForward ?? 0m)
                              + (values?.OrdinaryDepreciationPosted ?? 0m)
                              + (values?.UnplannedDepreciationPosted ?? 0m);

            var basis = new DepreciationBasis(
                acquisition,
                accumulated,
                row.ScrapValue ?? 0m,
                row.UsefulLifeYears,
                row.UsefulLifePeriods,
                row.DepreciationStartDate ?? row.CapitalizationDate!.Value,
                PeriodsPerYear: 12,
                Method: ParseMethod(row.DepreciationMethod),
                DeclineFactor: row.DeclineFactor ?? 1m,
                SwitchToStraightLine: row.ChangeMethodAtEnd is not null);

            var charge = DepreciationCalculator.Calculate(
                basis, period.PeriodStartDate, period.PeriodEndDate);

            plan.Add(new PlannedDepreciation(
                row.AssetNumber,
                row.AssetSubNumber,
                row.AreaCode,
                charge.Amount,
                charge.NetBookValueAfter,
                charge.IsFinalCharge,
                charge.Basis));
        }

        return plan;
    }

    private static DepreciationMethod ParseMethod(string method) => method switch
    {
        "DecliningBalance" => DepreciationMethod.DecliningBalance,
        "Immediate" => DepreciationMethod.Immediate,
        "Manual" => DepreciationMethod.Manual,
        _ => DepreciationMethod.StraightLine,
    };

    private async Task<(FiscalPeriod? Period, RuleViolation? Violation)> LoadPeriodAsync(
        DepreciationRunRequest request,
        CancellationToken cancellationToken)
    {
        var period = await (
            from p in context.Query<FiscalPeriod>().AsNoTracking()
            join company in context.Query<CompanyCode>()
                on p.FiscalYearVariantId equals company.FiscalYearVariantId
            where p.TenantId == TenantId
                  && company.CompanyCodeKey == request.CompanyCode
                  && p.FiscalYear == request.FiscalYear
                  && p.FiscalPeriodCode == request.FiscalPeriod
            select p).FirstOrDefaultAsync(cancellationToken);

        return period is null
            ? (null, new RuleViolation(
                AssetErrorCodes.PeriodUnknown,
                $"Period {request.FiscalPeriod}/{request.FiscalYear} is not defined for " +
                $"{request.CompanyCode}."))
            : (period, null);
    }

    private async Task<AssetSnapshot?> LoadAssetAsync(
        string companyCode,
        string assetNumber,
        int subNumber,
        CancellationToken cancellationToken)
    {
        var asset = await (
            from a in context.Query<Asset>()
            join company in context.Query<CompanyCode>() on a.CompanyCodeId equals company.Id
            where a.TenantId == TenantId
                  && company.CompanyCodeKey == companyCode
                  && a.AssetNumber == assetNumber
                  && a.AssetSubNumber == subNumber
            select a).FirstOrDefaultAsync(cancellationToken);

        if (asset is null)
        {
            return null;
        }

        var areas = await (
            from area in context.Query<AssetDepreciationArea>().AsNoTracking()
            join depreciationArea in context.Query<DepreciationArea>()
                on area.DepreciationAreaId equals depreciationArea.Id
            where area.TenantId == TenantId && area.AssetId == asset.Id && !area.IsDeactivated
            select new AssetAreaSnapshot(
                area.DepreciationAreaId,
                depreciationArea.DepreciationAreaCode,
                depreciationArea.PostsToGeneralLedger)
        ).ToListAsync(cancellationToken);

        var assetClass = await context.Query<AssetClass>()
            .AsNoTracking()
            .FirstAsync(c => c.Id == asset.AssetClassId, cancellationToken);

        var costCenter = asset.CostCenterId is null
            ? null
            : await context.Query<CostCenter>().AsNoTracking()
                .Where(c => c.Id == asset.CostCenterId)
                .Select(c => c.CostCenterCode)
                .FirstOrDefaultAsync(cancellationToken);

        var profitCenter = asset.ProfitCenterId is null
            ? null
            : await context.Query<ProfitCenter>().AsNoTracking()
                .Where(p => p.Id == asset.ProfitCenterId)
                .Select(p => p.ProfitCenterCode)
                .FirstOrDefaultAsync(cancellationToken);

        return new AssetSnapshot(asset, assetClass, areas, costCenter, profitCenter);
    }

    /// <summary>
    /// Resolves the four accounts an asset movement needs from the account
    /// determination of its class. Missing configuration is reported as a rule
    /// violation, not as a null reference at posting time.
    /// </summary>
    private async Task<AssetAccounts> ResolveAccountsAsync(
        AssetSnapshot asset,
        CancellationToken cancellationToken)
    {
        var modifier = asset.AssetClass.AccountDeterminationKey;

        var rules = await (
            from rule in context.Query<AccountDeterminationRule>().AsNoTracking()
            join account in context.Query<GLAccount>() on rule.DebitGLAccountId equals account.Id
            where rule.TenantId == TenantId && rule.AccountModifier == modifier
            select new { rule.TransactionKey, account.GLAccountCode }
        ).ToListAsync(cancellationToken);

        var missing = new List<RuleViolation>();

        string Account(string key)
        {
            var found = rules.FirstOrDefault(r => r.TransactionKey == key)?.GLAccountCode;
            if (found is null)
            {
                missing.Add(new RuleViolation(
                    AssetErrorCodes.AccountDeterminationMissing,
                    $"No account is configured for key {key} and determination {modifier}. " +
                    "Maintain cfg.AccountDeterminationRule before posting to this asset class."));
            }

            return found ?? string.Empty;
        }

        var accounts = new AssetAccounts
        {
            AcquisitionValue = Account(AssetTransactionKeys.AcquisitionValue),
            AccumulatedDepreciation = Account(AssetTransactionKeys.AccumulatedDepreciation),
            DepreciationExpense = Account(AssetTransactionKeys.DepreciationExpense),
            RetirementGain = Account(AssetTransactionKeys.RetirementGain),
            RetirementLoss = Account(AssetTransactionKeys.RetirementLoss),
            AcquisitionTransactionTypeId = await context.Query<AssetTransactionType>()
                .AsNoTracking()
                .Where(t => t.TenantId == TenantId && t.TransactionTypeCode == "100")
                .Select(t => t.Id)
                .FirstOrDefaultAsync(cancellationToken),
            Missing = missing,
        };

        return accounts;
    }

    private async Task<JournalEntryHeader?> LoadHeaderAsync(
        PostingResult posted,
        CancellationToken cancellationToken) =>
        await context.Query<JournalEntryHeader>()
            .AsNoTracking()
            .FirstOrDefaultAsync(
                h => h.TenantId == TenantId
                     && h.DocumentNumber == posted.DocumentNumber
                     && h.FiscalYear == posted.FiscalYear,
                cancellationToken);

    private async Task UpdateValuesAsync(
        long assetId,
        long areaId,
        short fiscalYear,
        string currency,
        decimal acquisitions,
        decimal depreciation,
        decimal retirements,
        DateTime now,
        CancellationToken cancellationToken)
    {
        var values = await context.Query<AssetValue>()
            .FirstOrDefaultAsync(
                v => v.TenantId == TenantId
                     && v.AssetId == assetId
                     && v.DepreciationAreaId == areaId
                     && v.FiscalYear == fiscalYear,
                cancellationToken);

        if (values is null)
        {
            values = new AssetValue
            {
                TenantId = TenantId,
                AssetId = assetId,
                DepreciationAreaId = areaId,
                FiscalYear = fiscalYear,
                CurrencyCode = currency,
                LastUpdatedAt = now,
            };

            context.Add(values);
        }

        values.CurrentYearAcquisitions += acquisitions;
        values.OrdinaryDepreciationPosted += depreciation;
        values.CurrentYearRetirements += retirements;
        values.NetBookValue =
            values.AcquisitionValueBroughtForward
            + values.CurrentYearAcquisitions
            - values.CurrentYearRetirements
            - values.AccumulatedDepreciationBroughtForward
            - values.OrdinaryDepreciationPosted
            - values.SpecialDepreciationPosted
            - values.UnplannedDepreciationPosted
            - values.ImpairmentPosted
            + values.WriteUpPosted;
        values.LastUpdatedAt = now;
    }

    private async Task<decimal> NetBookValueAsync(
        long assetId,
        long areaId,
        short fiscalYear,
        CancellationToken cancellationToken) =>
        await context.Query<AssetValue>()
            .AsNoTracking()
            .Where(v => v.AssetId == assetId
                        && v.DepreciationAreaId == areaId
                        && v.FiscalYear == fiscalYear)
            .Select(v => v.NetBookValue)
            .FirstOrDefaultAsync(cancellationToken);

    private static AssetPostingResult Failed(string code, string message, string field) =>
        new() { Violations = [new RuleViolation(code, message, field)] };

    private sealed record AssetSnapshot(
        Asset Asset,
        AssetClass AssetClass,
        List<AssetAreaSnapshot> Areas,
        string? CostCenter,
        string? ProfitCenter)
    {
        /// <summary>The area that posts to the ledger; depreciation follows it.</summary>
        public long BookAreaId =>
            Areas.FirstOrDefault(a => a.PostsToGeneralLedger == "RealTime")?.DepreciationAreaId
            ?? Areas[0].DepreciationAreaId;
    }

    private sealed record AssetAreaSnapshot(
        long DepreciationAreaId,
        string DepreciationArea,
        string PostsToGeneralLedger);

    private sealed class AssetAccounts
    {
        public required string AcquisitionValue { get; init; }

        public required string AccumulatedDepreciation { get; init; }

        public required string DepreciationExpense { get; init; }

        public required string RetirementGain { get; init; }

        public required string RetirementLoss { get; init; }

        public required long AcquisitionTransactionTypeId { get; init; }

        public required List<RuleViolation> Missing { get; init; }
    }
}
