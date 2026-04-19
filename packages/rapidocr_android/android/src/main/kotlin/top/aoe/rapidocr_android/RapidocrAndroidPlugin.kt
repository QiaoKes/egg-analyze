package top.aoe.rapidocr_android

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.os.Handler
import android.os.Looper
import com.benjaminwan.ocrlibrary.OcrEngine
import com.benjaminwan.ocrlibrary.Point
import com.benjaminwan.ocrlibrary.TextBlock
import io.flutter.embedding.engine.plugins.FlutterPlugin
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import io.flutter.plugin.common.MethodChannel.MethodCallHandler
import io.flutter.plugin.common.MethodChannel.Result
import java.util.concurrent.ExecutorService
import java.util.concurrent.Executors
import kotlin.math.max
import kotlin.math.min

class RapidocrAndroidPlugin : FlutterPlugin, MethodCallHandler {
    private val mainHandler = Handler(Looper.getMainLooper())
    private lateinit var channel: MethodChannel
    private lateinit var executor: ExecutorService
    private var applicationContext: Context? = null
    private var ocrEngine: OcrEngine? = null

    override fun onAttachedToEngine(binding: FlutterPlugin.FlutterPluginBinding) {
        applicationContext = binding.applicationContext
        executor = Executors.newSingleThreadExecutor()
        channel = MethodChannel(binding.binaryMessenger, "rapidocr_android/methods")
        channel.setMethodCallHandler(this)
    }

    override fun onMethodCall(call: MethodCall, result: Result) {
        when (call.method) {
            "initialize" -> {
                executor.execute {
                    runSafely(result) {
                        ensureEngine()
                        null
                    }
                }
            }

            "recognize" -> {
                val imageBytes = call.argument<ByteArray>("imageBytes")
                executor.execute {
                    runSafely(result) {
                        recognizeSingle(
                            imageBytes = imageBytes
                                ?: throw IllegalArgumentException("imageBytes is required"),
                            mode = call.argument<String>("mode"),
                        )
                    }
                }
            }

            "recognizeBatch" -> {
                val images = call.argument<List<ByteArray>>("images")
                executor.execute {
                    runSafely(result) {
                        (images ?: emptyList()).map { imageBytes ->
                            recognizeSingle(
                                imageBytes = imageBytes,
                                mode = call.argument<String>("mode"),
                            )
                        }
                    }
                }
            }

            "dispose" -> {
                executor.execute {
                    runSafely(result) {
                        ocrEngine = null
                        null
                    }
                }
            }

            else -> result.notImplemented()
        }
    }

    override fun onDetachedFromEngine(binding: FlutterPlugin.FlutterPluginBinding) {
        channel.setMethodCallHandler(null)
        ocrEngine = null
        applicationContext = null
        executor.shutdown()
    }

    private fun ensureEngine(): OcrEngine {
        ocrEngine?.let { return it }

        val context = applicationContext ?: error("RapidOCR Android context is unavailable")
        val engine = OcrEngine(context)
        engine.boxScoreThresh = 0.42f
        engine.boxThresh = 0.3f
        engine.unClipRatio = 1.6f
        engine.doAngle = true
        engine.mostAngle = true
        ocrEngine = engine
        return engine
    }

    private fun recognizeSingle(imageBytes: ByteArray, mode: String?): Map<String, Any?> {
        val bitmap = BitmapFactory.decodeByteArray(imageBytes, 0, imageBytes.size)
            ?: throw IllegalArgumentException("Unable to decode image bytes")

        val boxBitmap = Bitmap.createBitmap(
            bitmap.width,
            bitmap.height,
            Bitmap.Config.ARGB_8888,
        )

        val maxSideLen = calculateMaxSideLen(bitmap)
        val result = ensureEngine().detect(bitmap, boxBitmap, maxSideLen)
        bitmap.recycle()
        boxBitmap.recycle()

        return mapOf(
            "mode" to (mode ?: "full"),
            "lines" to result.textBlocks.map(::textBlockToMap),
        )
    }

    private fun calculateMaxSideLen(bitmap: Bitmap): Int {
        val longestEdge = max(bitmap.width, bitmap.height)
        return min(longestEdge, 2560)
    }

    private fun textBlockToMap(block: TextBlock): Map<String, Any> {
        val bounds = blockBounds(block.boxPoint)
        return mapOf(
            "text" to block.text.trim(),
            "left" to bounds.left,
            "top" to bounds.top,
            "width" to bounds.width,
            "height" to bounds.height,
        )
    }

    private fun blockBounds(points: List<Point>): Bounds {
        if (points.isEmpty()) {
            return Bounds(0.0, 0.0, 0.0, 0.0)
        }

        var minX = Int.MAX_VALUE
        var minY = Int.MAX_VALUE
        var maxX = Int.MIN_VALUE
        var maxY = Int.MIN_VALUE
        for (point in points) {
            minX = min(minX, point.x)
            minY = min(minY, point.y)
            maxX = max(maxX, point.x)
            maxY = max(maxY, point.y)
        }
        return Bounds(
            left = minX.toDouble(),
            top = minY.toDouble(),
            width = (maxX - minX).toDouble(),
            height = (maxY - minY).toDouble(),
        )
    }

    private fun runSafely(result: Result, block: () -> Any?) {
        try {
            val value = block()
            mainHandler.post {
                result.success(value)
            }
        } catch (error: Throwable) {
            mainHandler.post {
                result.error(
                    "rapidocr_android_error",
                    error.message ?: error.javaClass.simpleName,
                    null,
                )
            }
        }
    }
}

private data class Bounds(
    val left: Double,
    val top: Double,
    val width: Double,
    val height: Double,
)
