using System.Security.Cryptography;
using System.Text;

namespace HRModule.Api.Security;

/// <summary>
/// PBKDF2-HMAC-SHA256 password hashing. Parameters match the SQL seed
/// (100,000 iterations, 32-byte key, per-user salt) so seeded demo users
/// authenticate without re-hashing.
/// </summary>
public static class PasswordHasher
{
    private const int Iterations = 100_000;
    private const int KeyBytes = 32;
    private const int SaltBytes = 16;

    /// <summary>Hashes a new password, returning base64 (hash, salt).</summary>
    public static (string Hash, string Salt) Hash(string password)
    {
        var salt = RandomNumberGenerator.GetBytes(SaltBytes);
        var dk = Rfc2898DeriveBytes.Pbkdf2(
            Encoding.UTF8.GetBytes(password), salt, Iterations, HashAlgorithmName.SHA256, KeyBytes);
        return (Convert.ToBase64String(dk), Convert.ToBase64String(salt));
    }

    /// <summary>Verifies a password against a stored base64 hash + salt (constant time).</summary>
    public static bool Verify(string password, string hashB64, string saltB64)
    {
        var salt = Convert.FromBase64String(saltB64);
        var expected = Convert.FromBase64String(hashB64);
        var actual = Rfc2898DeriveBytes.Pbkdf2(
            Encoding.UTF8.GetBytes(password), salt, Iterations, HashAlgorithmName.SHA256, expected.Length);
        return CryptographicOperations.FixedTimeEquals(actual, expected);
    }
}
