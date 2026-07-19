package com.example.masque

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts

class MainActivity : ComponentActivity() {

    private lateinit var statusView: TextView
    private lateinit var connectBtn: Button
    private var connected = false

    private val statusReceiver = object : BroadcastReceiver() {
        override fun onReceive(c: Context?, i: Intent?) {
            val msg = i?.getStringExtra("msg") ?: return
            statusView.text = "Статус: $msg"
            connected = msg == "Подключено" || msg == "VPN активен"
            connectBtn.text = if (connected) "Отключить" else "Подключить"
        }
    }

    private val pickProfile = registerForActivityResult(
        ActivityResultContracts.GetContent()
    ) { uri ->
        if (uri == null) return@registerForActivityResult
        val text = contentResolver.openInputStream(uri)?.bufferedReader()?.use { it.readText() }
        if (text == null) {
            toast("Не удалось прочитать файл")
            return@registerForActivityResult
        }
        ProfileStore.import(this, text)
            .onSuccess {
                toast("Профиль импортирован")
                refresh()
            }
            .onFailure { toast("Ошибка профиля: ${it.message}") }
    }

    private val vpnPermission = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { res ->
        if (res.resultCode == Activity.RESULT_OK) {
            startService(Intent(this, MasqueVpnService::class.java).setAction(MasqueVpnService.ACTION_CONNECT))
        } else {
            toast("Разрешение VPN отклонено")
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        statusView = findViewById(R.id.status)
        connectBtn = findViewById(R.id.btnConnect)

        findViewById<Button>(R.id.btnImport).setOnClickListener {
            pickProfile.launch("*/*")
        }

        connectBtn.setOnClickListener {
            if (connected) {
                startService(Intent(this, MasqueVpnService::class.java).setAction(MasqueVpnService.ACTION_DISCONNECT))
            } else {
                if (!ProfileStore.isConfigured(this)) {
                    toast("Сначала импортируйте профиль")
                    return@setOnClickListener
                }
                val prep = VpnService.prepare(this)
                if (prep != null) vpnPermission.launch(prep)
                else startService(Intent(this, MasqueVpnService::class.java).setAction(MasqueVpnService.ACTION_CONNECT))
            }
        }

        refresh()
    }

    override fun onResume() {
        super.onResume()
        val filter = IntentFilter("com.example.masque.STATUS")
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(statusReceiver, filter, Context.RECEIVER_NOT_EXPORTED)
        } else {
            @Suppress("UnspecifiedRegisterReceiverFlag")
            registerReceiver(statusReceiver, filter)
        }
    }

    override fun onPause() {
        super.onPause()
        try {
            unregisterReceiver(statusReceiver)
        } catch (_: Exception) {
        }
    }

    private fun refresh() {
        val ok = ProfileStore.isConfigured(this)
        statusView.text = if (ok) "Статус: профиль готов" else "Статус: профиль не настроен"
    }

    private fun toast(m: String) = Toast.makeText(this, m, Toast.LENGTH_SHORT).show()
}
