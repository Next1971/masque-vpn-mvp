package com.example.masque

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import android.util.Log
import mobile.Callback
import mobile.Config
import mobile.Mobile
import mobile.Tunnel

class MasqueVpnService : VpnService() {

    companion object {
        const val TAG = "MasqueVpn"
        const val ACTION_CONNECT = "com.example.masque.CONNECT"
        const val ACTION_DISCONNECT = "com.example.masque.DISCONNECT"
        const val CHANNEL_ID = "masque_vpn"
        const val NOTIF_ID = 1

        const val TUN_ADDR_FALLBACK = "10.8.0.254"
        const val TUN_PREFIX = 24
        const val TUN_MTU = 1400
    }

    private var tunnel: Tunnel? = null
    private var pfd: ParcelFileDescriptor? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_DISCONNECT -> {
                stopVpn()
                return START_NOT_STICKY
            }
            else -> startVpn()
        }
        return START_STICKY
    }

    private fun startVpn() {
        val prof = ProfileStore.load(this)
        if (prof == null) {
            Log.e(TAG, "no profile configured")
            broadcast("Ошибка: профиль не настроен")
            stopSelf()
            return
        }

        startForeground(NOTIF_ID, buildNotification("Подключение…"))

        val builder = Builder()
            .setSession("MASQUE")
            .setMtu(TUN_MTU)
            .addAddress(TUN_ADDR_FALLBACK, TUN_PREFIX)
            .addRoute("0.0.0.0", 0)
            .addDnsServer(prof.dns)

        try {
            builder.addDisallowedApplication(packageName)
        } catch (e: Exception) {
            Log.w(TAG, "addDisallowedApplication: ${e.message}")
        }

        val iface = builder.establish()
        if (iface == null) {
            Log.e(TAG, "establish() returned null")
            broadcast("Ошибка: нет разрешения VPN")
            stopSelf()
            return
        }
        pfd = iface

        val cfg = Config().apply {
            server = prof.server
            serverName = prof.serverName
            caPath = prof.caPath
            certPath = prof.certPath
            keyPath = prof.keyPath
            mtu = TUN_MTU.toLong()
        }

        val cb = object : Callback {
            override fun onStatus(msg: String?) {
                Log.i(TAG, "status: $msg")
                broadcast(msg ?: "")
                updateNotification(msg ?: "Подключено")
            }

            override fun onError(msg: String?) {
                Log.e(TAG, "error: $msg")
                broadcast("Ошибка: $msg")
                stopVpn()
            }
        }

        try {
            val fd = iface.fd
            tunnel = Mobile.connect(cfg, fd.toLong(), cb)
            broadcast("Подключено")
            updateNotification("VPN активен")
        } catch (e: Exception) {
            Log.e(TAG, "connect failed", e)
            broadcast("Ошибка подключения: ${e.message}")
            stopVpn()
        }
    }

    private fun stopVpn() {
        try {
            tunnel?.stop()
        } catch (e: Exception) {
            Log.w(TAG, "tunnel.stop: ${e.message}")
        }
        tunnel = null

        try {
            pfd?.close()
        } catch (_: Exception) {
        }
        pfd = null

        broadcast("Отключено")
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    override fun onDestroy() {
        stopVpn()
        super.onDestroy()
    }

    private fun broadcast(msg: String) {
        sendBroadcast(Intent("com.example.masque.STATUS").putExtra("msg", msg).setPackage(packageName))
    }

    private fun ensureChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val nm = getSystemService(NotificationManager::class.java)
            if (nm.getNotificationChannel(CHANNEL_ID) == null) {
                nm.createNotificationChannel(
                    NotificationChannel(CHANNEL_ID, "MASQUE VPN", NotificationManager.IMPORTANCE_LOW)
                )
            }
        }
    }

    private fun buildNotification(text: String): Notification {
        ensureChannel()
        val pi = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        return Notification.Builder(this, CHANNEL_ID)
            .setContentTitle("MASQUE VPN")
            .setContentText(text)
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentIntent(pi)
            .setOngoing(true)
            .build()
    }

    private fun updateNotification(text: String) {
        val nm = getSystemService(NotificationManager::class.java)
        nm.notify(NOTIF_ID, buildNotification(text))
    }
}
