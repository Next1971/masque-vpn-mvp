package com.example.masque

import android.content.Context
import android.util.Log
import java.io.File

data class Profile(
    val server: String,
    val serverName: String,
    val dns: String,
    val caPath: String,
    val certPath: String,
    val keyPath: String,
)

object ProfileStore {
    private const val TAG = "MasqueProfile"
    private const val PREFS = "masque"
    private const val KEY_CONFIGURED = "configured"

    fun isConfigured(ctx: Context): Boolean =
        ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean(KEY_CONFIGURED, false) &&
            File(certsDir(ctx), "server.txt").exists()

    private fun certsDir(ctx: Context): File =
        File(ctx.filesDir, "certs").apply { mkdirs() }

    fun import(ctx: Context, content: String): Result<Unit> {
        return try {
            val server = extractValue(content, "address")
                ?: extractValue(content, "server")
                ?: return Result.failure(IllegalArgumentException("нет server.address"))
            val name = extractValue(content, "name") ?: server.substringBefore(":")
            val dns = extractValue(content, "dns") ?: "1.1.1.1"
            val ca = extractBlock(content, "ca") ?: return Result.failure(IllegalArgumentException("нет tls.ca"))
            val cert = extractBlock(content, "cert") ?: return Result.failure(IllegalArgumentException("нет tls.cert"))
            val key = extractBlock(content, "key") ?: return Result.failure(IllegalArgumentException("нет tls.key"))

            val dir = certsDir(ctx)
            File(dir, "ca.crt").writeText(ca.trim() + "\n")
            File(dir, "client.crt").writeText(cert.trim() + "\n")
            File(dir, "client.key").writeText(key.trim() + "\n")
            File(dir, "server.txt").writeText("$server\n$name\n$dns\n")

            ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .edit()
                .putBoolean(KEY_CONFIGURED, true)
                .apply()

            Log.i(TAG, "profile imported")
            Result.success(Unit)
        } catch (e: Exception) {
            Log.e(TAG, "import failed", e)
            Result.failure(e)
        }
    }

    fun load(ctx: Context): Profile? {
        val dir = certsDir(ctx)
        val meta = File(dir, "server.txt")
        if (!meta.exists()) return null
        val lines = meta.readLines()
        if (lines.size < 3) return null
        return Profile(
            server = lines[0].trim(),
            serverName = lines[1].trim(),
            dns = lines[2].trim(),
            caPath = File(dir, "ca.crt").absolutePath,
            certPath = File(dir, "client.crt").absolutePath,
            keyPath = File(dir, "client.key").absolutePath,
        )
    }

    private fun extractValue(content: String, key: String): String? {
        val re = Regex("""(?m)^\s*${Regex.escape(key)}\s*=\s*"([^"]*)"""")
        return re.find(content)?.groupValues?.get(1)
    }

    private fun extractBlock(content: String, key: String): String? {
        val re = Regex("""(?s)\b${Regex.escape(key)}\s*=\s*"{3}(.*?)"{3}""")
        return re.find(content)?.groupValues?.get(1)
    }
}
