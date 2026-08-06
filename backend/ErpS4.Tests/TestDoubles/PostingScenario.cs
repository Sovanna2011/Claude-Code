using ErpS4.Application.Posting;
using ErpS4.Application.Services;
using ErpS4.Database;
using ErpS4.Database.Entities;
using ErpS4.Domain;
using Microsoft.Extensions.Logging.Abstractions;

namespace ErpS4.Tests.TestDoubles;

/// <summary>
/// A minimal but complete configuration: one company code in USD, a chart of
/// accounts, an open period, the posting keys and accounts the tests use, and a
/// customer with a company code segment.
/// </summary>
/// <remarks>
/// Built once per test and mutated by the test that needs a variation - closing
/// the period, blocking an account - so each test states only what it is about.
/// </remarks>
public sealed class PostingScenario
{
    public const string CompanyCodeKey = "KH01";
    public const string LocalCurrency = "USD";
    public const string Revenue = "400000";
    public const string Expense = "500000";
    public const string Receivable = "120000";
    public const string Bank = "110000";
    public const string Customer = "1000000001";

    public PostingScenario()
    {
        Context = new InMemoryDataContext();

        Tenant = new Tenant { Id = 1, TenantCode = "100", Name = "Test", DefaultLanguage = "EN" };

        ChartOfAccounts = new ChartOfAccounts
        {
            TenantId = 1, ChartOfAccountsCode = "INT", Name = "International",
            MaintenanceLanguage = "EN", AccountNumberLength = 6, ChartType = "Operational",
        };

        FiscalYearVariant = new FiscalYearVariant
        {
            TenantId = 1, FiscalYearVariantCode = "K4", Name = "Calendar year",
            NumberOfPostingPeriods = 12, NumberOfSpecialPeriods = 4, IsCalendarYear = true,
        };

        PostingPeriodVariant = new PostingPeriodVariant
        {
            TenantId = 1, PostingPeriodVariantCode = "KH00", Name = "Group",
        };

        Context.Seed(Tenant).Seed(ChartOfAccounts).Seed(FiscalYearVariant).Seed(PostingPeriodVariant);

        CompanyCode = new CompanyCode
        {
            TenantId = 1,
            CompanyCodeKey = CompanyCodeKey,
            CompanyId = 1,
            Name = "Test company",
            CountryCode = "KH",
            LocalCurrencyCode = LocalCurrency,
            GroupCurrencyCode = LocalCurrency,
            LanguageCode = "EN",
            ChartOfAccountsId = ChartOfAccounts.Id,
            FiscalYearVariantId = FiscalYearVariant.Id,
            PostingPeriodVariantId = PostingPeriodVariant.Id,
            FieldStatusVariantId = 1,
            ControllingAreaId = 1,
            ProfitCenterMandatory = false,
            ValidFrom = new DateOnly(2000, 1, 1),
            ValidTo = new DateOnly(9999, 12, 31),
        };

        Ledger = new Ledger
        {
            TenantId = 1, LedgerCode = "0L", Name = "Leading", IsLeading = true,
            AccountingPrincipleId = 1, LedgerType = "Standard",
        };

        Period = new FiscalPeriod
        {
            TenantId = 1,
            FiscalYearVariantId = FiscalYearVariant.Id,
            FiscalYear = 2026,
            FiscalPeriodCode = 1,
            PeriodStartDate = new DateOnly(2026, 1, 1),
            PeriodEndDate = new DateOnly(2026, 1, 31),
            PeriodStatus = "Open",
        };

        PeriodControl = new PostingPeriodControl
        {
            TenantId = 1,
            PostingPeriodVariantId = PostingPeriodVariant.Id,
            AccountType = "+",
            FromAccount = "",
            ToAccount = "ZZZZZZZZZZ",
            FromPeriod1 = 1, FromYear1 = 2026, ToPeriod1 = 12, ToYear1 = 2026,
        };

        DocumentType = new DocumentType
        {
            TenantId = 1, DocumentTypeCode = "SA", Name = "G/L document",
            NumberRangeObjectId = 1, NumberRangeCode = "01",
            AllowGLAccounts = true, AllowCustomerAccounts = true, AllowVendorAccounts = true,
        };

        Context.Seed(CompanyCode).Seed(Ledger).Seed(Period).Seed(PeriodControl).Seed(DocumentType);

        // Payment document types, so clearing can post through the same engine.
        Context.Seed(
            new DocumentType
            {
                TenantId = 1, DocumentTypeCode = "DZ", Name = "Customer payment",
                NumberRangeObjectId = 1, NumberRangeCode = "03",
                AllowGLAccounts = true, AllowCustomerAccounts = true,
            },
            new DocumentType
            {
                TenantId = 1, DocumentTypeCode = "KZ", Name = "Supplier payment",
                NumberRangeObjectId = 1, NumberRangeCode = "05",
                AllowGLAccounts = true, AllowVendorAccounts = true,
            });

        Context.Seed(
            new Currency { TenantId = 1, CurrencyCode = "USD", IsoCode = "USD", Name = "US Dollar", DecimalPlaces = 2 },
            new Currency { TenantId = 1, CurrencyCode = "KHR", IsoCode = "KHR", Name = "Riel", DecimalPlaces = 0 });

        Context.Seed(
            new PostingKey { TenantId = 1, PostingKeyCode = "40", Name = "Debit G/L", DebitCreditIndicator = "S", AccountType = "S" },
            new PostingKey { TenantId = 1, PostingKeyCode = "50", Name = "Credit G/L", DebitCreditIndicator = "H", AccountType = "S" },
            new PostingKey { TenantId = 1, PostingKeyCode = "01", Name = "Customer invoice", DebitCreditIndicator = "S", AccountType = "D" },
            new PostingKey { TenantId = 1, PostingKeyCode = "11", Name = "Customer credit memo", DebitCreditIndicator = "H", AccountType = "D" },
            new PostingKey { TenantId = 1, PostingKeyCode = "15", Name = "Incoming payment", DebitCreditIndicator = "H", AccountType = "D", PaymentTransaction = true },
            new PostingKey { TenantId = 1, PostingKeyCode = "25", Name = "Outgoing payment", DebitCreditIndicator = "S", AccountType = "K", PaymentTransaction = true });

        RevenueAccount = SeedAccount(Revenue, isReconciliation: false);
        ExpenseAccount = SeedAccount(Expense, isReconciliation: false);
        BankAccount = SeedAccount(Bank, isReconciliation: false);
        ReceivableAccount = SeedAccount(Receivable, isReconciliation: true, reconciliationType: "D");

        Partner = new BusinessPartner
        {
            TenantId = 1, PartnerNumber = Customer, PartnerCategory = "2",
            BusinessPartnerGroupId = 1, FullName = "Angkor Distribution",
            LanguageCode = "EN", Status = "Active",
            ValidFrom = new DateOnly(2000, 1, 1), ValidTo = new DateOnly(9999, 12, 31),
        };

        Context.Seed(Partner);

        PartnerSegment = new BusinessPartnerCompanyCode
        {
            TenantId = 1,
            BusinessPartnerId = Partner.Id,
            CompanyCodeId = CompanyCode.Id,
            RoleCategory = "Customer",
            ReconciliationGLAccountId = ReceivableAccount.Id,
        };

        Context.Seed(PartnerSegment);

        Clock = new FixedTimeProvider(new DateTimeOffset(2026, 1, 20, 9, 0, 0, TimeSpan.Zero));
        NumberRanges = new FakeNumberRangeService();
        CurrencyConverter = new FakeCurrencyConverter();
    }

    public InMemoryDataContext Context { get; }

    public Tenant Tenant { get; }

    public ChartOfAccounts ChartOfAccounts { get; }

    public FiscalYearVariant FiscalYearVariant { get; }

    public PostingPeriodVariant PostingPeriodVariant { get; }

    public CompanyCode CompanyCode { get; }

    public Ledger Ledger { get; }

    public FiscalPeriod Period { get; }

    public PostingPeriodControl PeriodControl { get; }

    public DocumentType DocumentType { get; }

    public GLAccount RevenueAccount { get; }

    public GLAccount ExpenseAccount { get; }

    public GLAccount BankAccount { get; }

    public GLAccount ReceivableAccount { get; }

    public BusinessPartner Partner { get; }

    public BusinessPartnerCompanyCode PartnerSegment { get; }

    public FixedTimeProvider Clock { get; }

    public FakeNumberRangeService NumberRanges { get; }

    public FakeCurrencyConverter CurrencyConverter { get; }

    /// <summary>An engine acting as a named user, for maker-checker tests.</summary>
    public PostingEngine CreateEngineAs(string userName) =>
        new(Context,
            NumberRanges,
            CurrencyConverter,
            new FixedTenantProvider(1),
            new FixedCurrentUser(userName),
            Clock,
            NullLogger<PostingEngine>.Instance);

    public PostingEngine CreateEngine() =>
        new(Context,
            NumberRanges,
            CurrencyConverter,
            new FixedTenantProvider(1),
            new FixedCurrentUser("tester"),
            Clock,
            NullLogger<PostingEngine>.Instance);

    /// <summary>A balanced two-line G/L document: expense against revenue.</summary>
    public static JournalEntryDraft BalancedDraft(decimal amount = 1000m, string currency = LocalCurrency) =>
        new JournalEntryDraft(
                CompanyCodeKey, "SA",
                new DateOnly(2026, 1, 20), new DateOnly(2026, 1, 20), currency)
            .AddLine(new JournalEntryDraftLine("40", Expense, new Money(amount, currency))
            {
                Text = "Expense",
            })
            .AddLine(new JournalEntryDraftLine("50", Revenue, new Money(-amount, currency))
            {
                Text = "Revenue",
            });

    private GLAccount SeedAccount(
        string number,
        bool isReconciliation,
        string? reconciliationType = null)
    {
        var account = new GLAccount
        {
            TenantId = 1,
            ChartOfAccountsId = ChartOfAccounts.Id,
            GLAccountCode = number,
            AccountGroupId = 1,
            AccountType = isReconciliation ? "BalanceSheet" : "PrimaryCost",
            IsReconciliationAccount = isReconciliation,
            ReconciliationAccountType = reconciliationType,
        };

        Context.Seed(account);
        Context.Seed(new GLAccountCompanyCode
        {
            TenantId = 1,
            GLAccountId = account.Id,
            CompanyCodeId = CompanyCode.Id,
            AccountCurrencyCode = LocalCurrency,
            FieldStatusGroupId = 1,
            IsLineItemDisplay = true,
        });

        return account;
    }
}

/// <summary>Deterministic clock.</summary>
public sealed class FixedTimeProvider(DateTimeOffset now) : TimeProvider
{
    public DateTimeOffset Now { get; set; } = now;

    public override DateTimeOffset GetUtcNow() => Now;
}

/// <summary>Hands out predictable document numbers and records what was asked for.</summary>
public sealed class FakeNumberRangeService : INumberRangeService
{
    private int _counter;

    public List<string> Issued { get; } = [];

    public Task<string> NextAsync(
        string numberRangeObject,
        string numberRangeCode,
        long? companyCodeId,
        short fiscalYear,
        string? documentType = null,
        CancellationToken cancellationToken = default)
    {
        var number = $"TST-{fiscalYear}-{documentType ?? "SA"}-{++_counter:0000000000}";
        Issued.Add(number);
        return Task.FromResult(number);
    }
}

/// <summary>USD is the base; KHR is pegged at 4,100 to keep arithmetic obvious.</summary>
public sealed class FakeCurrencyConverter : ICurrencyConverter
{
    private readonly Dictionary<(string From, string To), decimal> _rates = new()
    {
        [("USD", "KHR")] = 4100m,
        [("KHR", "USD")] = 1m / 4100m,
    };

    private readonly Dictionary<string, int> _decimals = new(StringComparer.Ordinal)
    {
        ["USD"] = 2,
        ["KHR"] = 0,
        ["THB"] = 2,
    };

    public Task<CurrencyConversion?> ConvertAsync(
        Money amount,
        string toCurrency,
        DateOnly date,
        string rateType = "M",
        CancellationToken cancellationToken = default)
    {
        if (string.Equals(amount.Currency, toCurrency, StringComparison.Ordinal))
        {
            return Task.FromResult<CurrencyConversion?>(
                new CurrencyConversion(amount, 1m, false));
        }

        if (!_rates.TryGetValue((amount.Currency, toCurrency), out var rate))
        {
            return Task.FromResult<CurrencyConversion?>(null);
        }

        return Task.FromResult<CurrencyConversion?>(new CurrencyConversion(
            new Money(amount.Amount * rate, toCurrency), rate, false));
    }

    public Task<int> GetDecimalsAsync(string currency, CancellationToken cancellationToken = default) =>
        Task.FromResult(_decimals.GetValueOrDefault(currency, 2));
}
