using ErpS4.Domain;
using FluentValidation;

namespace ErpS4.Application.Posting;

/// <summary>
/// Command side of the journal entry (FB50). Commands change data; the query
/// side reads from optimised read models and never goes through this path.
/// </summary>
public sealed record PostJournalEntryCommand
{
    public required string CompanyCode { get; init; }

    public required string DocumentType { get; init; }

    public required DateOnly DocumentDate { get; init; }

    public required DateOnly PostingDate { get; init; }

    public required string CurrencyCode { get; init; }

    public string? ReferenceDocumentNumber { get; init; }

    public string? HeaderText { get; init; }

    public required IReadOnlyList<PostJournalEntryLine> Lines { get; init; }

    /// <summary>Validate and calculate without writing (the Simulate button).</summary>
    public bool Simulate { get; init; }

    /// <summary>Sent as the Idempotency-Key header; a retry returns the first result.</summary>
    public Guid? IdempotencyKey { get; init; }

    public string SourceModule { get; init; } = "FI";

    public string? TransactionCode { get; init; }
}

/// <param name="Amount">Signed: positive debit, negative credit.</param>
public sealed record PostJournalEntryLine(string PostingKey, string Account, decimal Amount)
{
    public string? BusinessPartner { get; init; }

    public string? CostCenter { get; init; }

    public string? ProfitCenter { get; init; }

    public string? InternalOrder { get; init; }

    public string? Segment { get; init; }

    public string? TaxCode { get; init; }

    public decimal? TaxAmount { get; init; }

    public string? Assignment { get; init; }

    public string? Text { get; init; }

    public string? PaymentTerms { get; init; }

    public DateOnly? BaselineDate { get; init; }

    public string? PartnerCompanyCode { get; init; }
}

/// <summary>
/// Request-shape validation only. Business rules that need configuration -
/// period open, account postable, reconciliation accounts - belong to the
/// posting engine, which is the single place they are enforced for every
/// caller.
/// </summary>
public sealed class PostJournalEntryCommandValidator : AbstractValidator<PostJournalEntryCommand>
{
    public PostJournalEntryCommandValidator()
    {
        RuleFor(c => c.CompanyCode).NotEmpty().MaximumLength(4);
        RuleFor(c => c.DocumentType).NotEmpty().MaximumLength(2);
        RuleFor(c => c.CurrencyCode).NotEmpty().Length(3, 5);
        RuleFor(c => c.DocumentDate).NotEqual(default(DateOnly));
        RuleFor(c => c.PostingDate).NotEqual(default(DateOnly));
        RuleFor(c => c.HeaderText).MaximumLength(255);
        RuleFor(c => c.ReferenceDocumentNumber).MaximumLength(20);
        RuleFor(c => c.SourceModule).NotEmpty().MaximumLength(10);

        RuleFor(c => c.Lines)
            .NotNull()
            .Must(lines => lines is { Count: >= 2 })
            .WithMessage("A document needs at least two lines.");

        RuleForEach(c => c.Lines).ChildRules(line =>
        {
            line.RuleFor(l => l.PostingKey).NotEmpty().MaximumLength(2);
            line.RuleFor(l => l.Account).NotEmpty().MaximumLength(10);
            line.RuleFor(l => l.Amount).NotEqual(0m).WithMessage("A line amount cannot be zero.");
            line.RuleFor(l => l.Text).MaximumLength(255);
            line.RuleFor(l => l.Assignment).MaximumLength(18);
            line.RuleFor(l => l.TaxCode).MaximumLength(2);
        });
    }
}

/// <summary>Turns the command into a domain draft and hands it to the engine.</summary>
public sealed class PostJournalEntryHandler(
    IPostingEngine postingEngine,
    IValidator<PostJournalEntryCommand> validator)
{
    public async Task<PostingResult> HandleAsync(
        PostJournalEntryCommand command,
        CancellationToken cancellationToken = default)
    {
        var validation = await validator.ValidateAsync(command, cancellationToken);
        if (!validation.IsValid)
        {
            return PostingResult.Rejected(validation.Errors
                .Select(e => new PostingError("REQUEST.INVALID", e.ErrorMessage, e.PropertyName))
                .ToList());
        }

        var draft = new JournalEntryDraft(
            command.CompanyCode,
            command.DocumentType,
            command.DocumentDate,
            command.PostingDate,
            command.CurrencyCode)
        {
            HeaderText = command.HeaderText,
            ReferenceDocumentNumber = command.ReferenceDocumentNumber,
        };

        foreach (var line in command.Lines)
        {
            draft.AddLine(new JournalEntryDraftLine(
                line.PostingKey,
                line.Account,
                new Money(line.Amount, command.CurrencyCode))
            {
                BusinessPartner = line.BusinessPartner,
                CostCenter = line.CostCenter,
                ProfitCenter = line.ProfitCenter,
                InternalOrder = line.InternalOrder,
                Segment = line.Segment,
                TaxCode = line.TaxCode,
                TaxAmount = line.TaxAmount is null
                    ? null
                    : new Money(line.TaxAmount.Value, command.CurrencyCode),
                Assignment = line.Assignment,
                Text = line.Text,
                PaymentTerms = line.PaymentTerms,
                BaselineDate = line.BaselineDate,
                PartnerCompanyCode = line.PartnerCompanyCode,
            });
        }

        return await postingEngine.PostAsync(
            new PostingRequest(
                draft,
                command.SourceModule,
                command.TransactionCode,
                command.IdempotencyKey,
                command.Simulate),
            cancellationToken);
    }
}
